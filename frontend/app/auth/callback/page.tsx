"use client";

import { useEffect, Suspense } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import toast from "react-hot-toast";

function AuthCallbackHandler() {
  const router = useRouter();
  const searchParams = useSearchParams();

  useEffect(() => {
    const token = searchParams.get("token");
    const refresh = searchParams.get("refresh");

    if (token && refresh) {
      localStorage.setItem("accessToken", token);
      localStorage.setItem("refreshToken", refresh);
      toast.success("Successfully logged in!");
      
      // Let AuthProvider state update and reload
      window.location.href = "/dashboard";
    } else {
      toast.error("Authentication failed. Please try again.");
      router.push("/login");
    }
  }, [searchParams, router]);

  return (
    <div className="min-h-screen bg-slate-950 flex flex-col items-center justify-center text-slate-300">
      <div className="relative flex items-center justify-center mb-4">
        <div className="w-12 h-12 border-4 border-indigo-500/20 border-t-indigo-500 rounded-full animate-spin" />
      </div>
      <p className="text-sm font-light">Completing secure authentication callback...</p>
    </div>
  );
}

export default function AuthCallbackPage() {
  return (
    <Suspense fallback={
      <div className="min-h-screen bg-slate-950 flex flex-col items-center justify-center text-slate-300">
        <p className="text-sm font-light">Loading callback details...</p>
      </div>
    }>
      <AuthCallbackHandler />
    </Suspense>
  );
}
