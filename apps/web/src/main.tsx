import { QueryClientProvider } from "@tanstack/react-query";
import { RouterProvider } from "@tanstack/react-router";
import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { Toaster } from "sonner";
import { TooltipProvider } from "@/components/ui/tooltip";
import { createQueryClient } from "./app/query-client";
import { createAppRouter } from "./app/router";
import "./index.css";

const queryClient = createQueryClient();
const router = createAppRouter(queryClient);

const root = document.getElementById("root");
if (!root) throw new Error("#root missing from index.html");

createRoot(root).render(
  <StrictMode>
    <QueryClientProvider client={queryClient}>
      <TooltipProvider delayDuration={400}>
        <RouterProvider router={router} />
        <Toaster position="bottom-right" closeButton richColors={false} />
      </TooltipProvider>
    </QueryClientProvider>
  </StrictMode>,
);
