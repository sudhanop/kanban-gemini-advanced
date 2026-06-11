"use client";

import { useState } from "react";
import { useAuth } from "@/components/providers/AuthProvider";
import { KanbanSquare } from "lucide-react";
import axios from "axios";
import toast from "react-hot-toast";

export default function LoginPage() {
  const { login, loading } = useAuth();
  const [email, setEmail] = useState("");
  const [name, setName] = useState("");
  const [demoLoading, setDemoLoading] = useState(false);

  const handleDemoLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!email) {
      toast.error("Please enter a valid email address.");
      return;
    }

    setDemoLoading(true);
    try {
      const apiURL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080/api";
      const res = await axios.post(`${apiURL}/auth/bypass`, {
        email,
        name: name || "Demo User",
      });

      const { access_token, refresh_token } = res.data;
      if (access_token && refresh_token) {
        localStorage.setItem("accessToken", access_token);
        localStorage.setItem("refreshToken", refresh_token);
        toast.success("Successfully logged in!");
        window.location.href = "/dashboard";
      } else {
        toast.error("Authentication failed. Please try again.");
      }
    } catch (err: any) {
      console.error(err);
      toast.error(err.response?.data?.error || "Failed to connect to backend server.");
    } finally {
      setDemoLoading(false);
    }
  };

  return (
    <div className="relative min-h-screen bg-slate-950 text-slate-100 flex flex-col items-center justify-center overflow-y-auto py-12 px-4 font-sans">
      {/* Background glow effects */}
      <div className="absolute w-[400px] h-[400px] bg-indigo-900/10 rounded-full blur-[100px] top-1/4 left-1/4 pointer-events-none" />
      <div className="absolute w-[400px] h-[400px] bg-purple-900/10 rounded-full blur-[100px] bottom-1/4 right-1/4 pointer-events-none" />

      {/* Grid Pattern overlay */}
      <div className="absolute inset-0 bg-[linear-gradient(to_right,#0f172a_1px,transparent_1px),linear-gradient(to_bottom,#0f172a_1px,transparent_1px)] bg-[size:4rem_4rem] [mask-image:radial-gradient(ellipse_60%_50%_at_50%_50%,#000_70%,transparent_100%)] pointer-events-none opacity-30" />

      <div className="relative w-full max-w-md p-8 md:p-10 rounded-3xl border border-slate-900 bg-slate-950/40 backdrop-blur-xl shadow-2xl flex flex-col items-center text-center z-10">
        <div className="mb-6 p-3 bg-indigo-600/10 rounded-2xl border border-indigo-500/20 text-indigo-400">
          <KanbanSquare className="w-8 h-8" />
        </div>

        <h1 className="text-2xl font-bold tracking-tight text-white mb-2 font-outfit">Welcome to FlowBoard</h1>
        <p className="text-slate-400 text-sm font-light mb-8">
          The premium workflow and project management system powered by Gemini AI.
        </p>

        {/* Google OAuth Login */}
        <button
          onClick={login}
          disabled={loading || demoLoading}
          className="w-full flex items-center justify-center gap-3 px-5 py-4 border border-slate-800 bg-slate-900 hover:bg-slate-850 hover:border-slate-700 text-slate-100 hover:text-white rounded-2xl font-medium shadow-md transition duration-200"
        >
          {/* Custom inline Google Logo icon */}
          <svg className="w-5 h-5" viewBox="0 0 24 24" width="24" height="24">
            <path
              fill="#EA4335"
              d="M12 5.04c1.66 0 3.2.57 4.38 1.69l3.27-3.27C17.67 1.6 15.02 1 12 1 7.35 1 3.37 3.67 1.39 7.56l3.85 2.99C6.15 7.15 8.85 5.04 12 5.04z"
            />
            <path
              fill="#4285F4"
              d="M23.49 12.27c0-.81-.07-1.59-.2-2.34H12v4.44h6.45c-.28 1.48-1.11 2.74-2.37 3.58l3.69 2.87c2.16-1.99 3.42-4.92 3.42-8.55z"
            />
            <path
              fill="#FBBC05"
              d="M5.24 14.55c-.24-.72-.38-1.5-.38-2.3s.14-1.58.38-2.3L1.39 7.56C.5 9.35 0 11.33 0 13.43s.5 4.08 1.39 5.87l3.85-2.99z"
            />
            <path
              fill="#34A853"
              d="M12 23c3.24 0 5.97-1.07 7.96-2.91l-3.69-2.87c-1.02.68-2.33 1.09-4.27 1.09-3.15 0-5.85-2.11-6.76-5.51L1.39 16.44C3.37 20.33 7.35 23 12 23z"
            />
          </svg>
          Sign in with Google
        </button>

        {/* Divider */}
        <div className="relative my-6 w-full flex items-center justify-center">
          <div className="border-t border-slate-900 w-full" />
          <span className="absolute bg-slate-950 px-3 text-xs text-slate-500 uppercase tracking-widest">or</span>
        </div>

        {/* Demo Login Form */}
        <form onSubmit={handleDemoLogin} className="w-full flex flex-col gap-4">
          <div className="flex flex-col text-left gap-1.5">
            <label htmlFor="email" className="text-xs font-semibold text-slate-400 tracking-wide uppercase">Email Address</label>
            <input
              id="email"
              type="email"
              placeholder="e.g., user@example.com"
              required
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              className="w-full px-4 py-3 rounded-2xl border border-slate-900 bg-slate-950 hover:border-slate-800 focus:border-indigo-500 focus:outline-none text-sm text-slate-100 placeholder-slate-600 transition"
            />
          </div>

          <div className="flex flex-col text-left gap-1.5">
            <label htmlFor="name" className="text-xs font-semibold text-slate-400 tracking-wide uppercase">Your Name (Optional)</label>
            <input
              id="name"
              type="text"
              placeholder="e.g., Demo Developer"
              value={name}
              onChange={(e) => setName(e.target.value)}
              className="w-full px-4 py-3 rounded-2xl border border-slate-900 bg-slate-950 hover:border-slate-800 focus:border-indigo-500 focus:outline-none text-sm text-slate-100 placeholder-slate-600 transition"
            />
          </div>

          <button
            type="submit"
            disabled={demoLoading || loading}
            className="w-full mt-2 flex items-center justify-center gap-2 px-5 py-3.5 bg-indigo-600 hover:bg-indigo-550 disabled:opacity-50 text-white rounded-2xl font-semibold shadow-lg shadow-indigo-600/10 hover:shadow-indigo-500/20 transition duration-200"
          >
            {demoLoading ? (
              <div className="w-5 h-5 border-2 border-white/20 border-t-white rounded-full animate-spin" />
            ) : (
              "Sign In Instantly"
            )}
          </button>
        </form>

        <div className="mt-8 text-xs text-slate-500 font-light max-w-xs">
          By signing in, you agree to our Terms of Service and Privacy Policy. FlowBoard stores your Google email and avatar.
        </div>
      </div>
    </div>
  );
}
