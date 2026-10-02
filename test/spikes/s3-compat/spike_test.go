//go:build spike

// Package s3compat is the ADR-017 spike: it holds each candidate S3-compatible
// server to the browser direct-upload contract OpsGrid needs (and that
// Cloudflare R2 provides). Run with ./run.sh.
package s3compat

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
)

const (
	origin = "http://localhost:5173"
	bucket = "spike"
)

type target struct {
	name, endpoint, region string
	createBucket           bool
}

var targets = []target{
	{name: "garage", endpoint: "http://localhost:13900", region: "garage"},
	{name: "seaweedfs", endpoint: "http://localhost:18333", region: "us-east-1", createBucket: true},
}

func client(t target) *s3.Client {
	return s3.New(s3.Options{
		BaseEndpoint: aws.String(t.endpoint),
		Region:       t.region,
		UsePathStyle: true,
		Credentials: credentials.NewStaticCredentialsProvider(
			os.Getenv("S3_SPIKE_ACCESS"), os.Getenv("S3_SPIKE_SECRET"), ""),
	})
}

func do(t *testing.T, method, url string, body []byte, headers map[string]string) *http.Response {
	t.Helper()
	req, err := http.NewRequestWithContext(context.Background(), method, url, bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.ContentLength = int64(len(body))
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	return resp
}

func TestDirectUploadContract(t *testing.T) {
	if os.Getenv("S3_SPIKE_ACCESS") == "" {
		t.Skip("run via test/spikes/s3-compat/run.sh")
	}
	for _, tg := range targets {
		t.Run(tg.name, func(t *testing.T) {
			ctx := context.Background()
			c := client(tg)
			presign := s3.NewPresignClient(c)

			t.Run("setup bucket and CORS", func(t *testing.T) {
				if tg.createBucket {
					_, err := c.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: aws.String(bucket)})
					var apiErr smithy.APIError
					if err != nil && !(errors.As(err, &apiErr) && strings.Contains(apiErr.ErrorCode(), "BucketAlready")) {
						t.Fatalf("create bucket: %v", err)
					}
				}
				_, err := c.PutBucketCors(ctx, &s3.PutBucketCorsInput{
					Bucket: aws.String(bucket),
					CORSConfiguration: &types.CORSConfiguration{CORSRules: []types.CORSRule{{
						AllowedOrigins: []string{origin},
						AllowedMethods: []string{"PUT", "GET"},
						AllowedHeaders: []string{"content-type", "content-length"},
						ExposeHeaders:  []string{"ETag"},
						MaxAgeSeconds:  aws.Int32(600),
					}}},
				})
				if err != nil {
					t.Fatalf("PutBucketCors: %v", err)
				}
			})

			const size = 1024
			body := bytes.Repeat([]byte("x"), size)
			key := fmt.Sprintf("org/a/wo/b/%d.jpg", time.Now().UnixNano())
			put, err := presign.PresignPutObject(ctx, &s3.PutObjectInput{
				Bucket: aws.String(bucket), Key: aws.String(key),
				ContentType: aws.String("image/jpeg"), ContentLength: aws.Int64(size),
			}, s3.WithPresignExpires(5*time.Minute))
			if err != nil {
				t.Fatal(err)
			}

			t.Run("CORS preflight allows the app origin", func(t *testing.T) {
				resp := do(t, http.MethodOptions, put.URL, nil, map[string]string{
					"Origin":                         origin,
					"Access-Control-Request-Method":  "PUT",
					"Access-Control-Request-Headers": "content-type",
				})
				if resp.StatusCode >= 300 || resp.Header.Get("Access-Control-Allow-Origin") == "" {
					t.Fatalf("preflight: status %d, allow-origin %q", resp.StatusCode, resp.Header.Get("Access-Control-Allow-Origin"))
				}
			})

			t.Run("CORS preflight rejects other origins", func(t *testing.T) {
				resp := do(t, http.MethodOptions, put.URL, nil, map[string]string{
					"Origin":                        "https://evil.example",
					"Access-Control-Request-Method": "PUT",
				})
				if resp.StatusCode < 300 && resp.Header.Get("Access-Control-Allow-Origin") != "" {
					t.Fatalf("foreign origin allowed: %q", resp.Header.Get("Access-Control-Allow-Origin"))
				}
			})

			t.Run("presigned PUT from the browser origin succeeds", func(t *testing.T) {
				resp := do(t, http.MethodPut, put.URL, body, map[string]string{
					"Origin": origin, "Content-Type": "image/jpeg",
				})
				if resp.StatusCode != http.StatusOK {
					t.Fatalf("PUT status %d", resp.StatusCode)
				}
				if resp.Header.Get("Access-Control-Allow-Origin") == "" {
					t.Error("PUT response lacks Access-Control-Allow-Origin (browser would block reading it)")
				}
			})

			t.Run("signed content-type is enforced", func(t *testing.T) {
				resp := do(t, http.MethodPut, put.URL, body, map[string]string{"Content-Type": "text/html"})
				if resp.StatusCode == http.StatusOK {
					t.Fatal("upload with a different content-type was accepted")
				}
			})

			t.Run("signed content-length is enforced", func(t *testing.T) {
				resp := do(t, http.MethodPut, put.URL, bytes.Repeat([]byte("x"), 4*size), map[string]string{"Content-Type": "image/jpeg"})
				if resp.StatusCode == http.StatusOK {
					t.Fatal("oversized upload was accepted")
				}
			})

			t.Run("presigned GET works then expires", func(t *testing.T) {
				get, err := presign.PresignGetObject(ctx, &s3.GetObjectInput{Bucket: aws.String(bucket), Key: aws.String(key)},
					s3.WithPresignExpires(2*time.Second))
				if err != nil {
					t.Fatal(err)
				}
				if resp := do(t, http.MethodGet, get.URL, nil, nil); resp.StatusCode != http.StatusOK {
					t.Fatalf("fresh GET status %d", resp.StatusCode)
				}
				time.Sleep(3 * time.Second)
				if resp := do(t, http.MethodGet, get.URL, nil, nil); resp.StatusCode == http.StatusOK {
					t.Fatal("expired presigned GET still works")
				}
			})

			t.Run("HEAD reports size and type (finalize step)", func(t *testing.T) {
				out, err := c.HeadObject(ctx, &s3.HeadObjectInput{Bucket: aws.String(bucket), Key: aws.String(key)})
				if err != nil {
					t.Fatal(err)
				}
				if aws.ToInt64(out.ContentLength) != size || aws.ToString(out.ContentType) != "image/jpeg" {
					t.Fatalf("HEAD = %d bytes, %q", aws.ToInt64(out.ContentLength), aws.ToString(out.ContentType))
				}
			})
		})
	}
}
