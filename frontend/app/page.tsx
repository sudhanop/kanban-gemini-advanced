"use client";

import Link from "next/link";
import { useAuth } from "@/components/providers/AuthProvider";
import {
  KanbanSquare,
  Sparkles,
  Zap,
  Shield,
  Activity,
  Calendar,
  Layers,
  ArrowRight,
  TrendingUp,
  Mail,
  Video,
} from "lucide-react";

export default function LandingPage() {
  const { isAuthenticated, login } = useAuth();

  const features = [
    {
      icon: <KanbanSquare className="w-6 h-6 text-indigo-400" />,
      title: "Interactive Kanban",
      desc: "Full drag and drop boards, task metadata customization, and live status column configurations.",
    },
    {
      icon: <Sparkles className="w-6 h-6 text-purple-400" />,
      title: "Gemini AI Assistant",
      desc: "Automated task suggestions, priority recommendation gates, board status analysis, and completion prediction.",
    },
    {
      icon: <Layers className="w-6 h-6 text-pink-400" />,
      title: "Sprint Management",
      desc: "Plan sprints, assign sprint scopes, view active backlogs, and track progress using automated burndown charts.",
    },
    {
      icon: <Activity className="w-6 h-6 text-emerald-400" />,
      title: "Velocity & Analytics",
      desc: "Real-time board metric aggregations, workload maps, user velocity tracking, and workload insights.",
    },
    {
      icon: <Calendar className="w-6 h-6 text-amber-400" />,
      title: "Task Timeline & Gantt",
      desc: "Visually schedule milestones, task dependencies, track start and end schedules on a timeline slider.",
    },
    {
      icon: <Video className="w-6 h-6 text-sky-400" />,
      title: "Jitsi & Meet Integrations",
      desc: "Create and attach instant video meeting rooms to workspaces for immediate group standups.",
    },
  ];

  return (
    <div className="relative min-h-screen bg-slate-950 text-slate-100 overflow-hidden font-sans">
      {/* Background Glows */}
      <div className="absolute top-[-10%] left-[-10%] w-[50%] h-[50%] bg-indigo-900/20 rounded-full blur-[120px] pointer-events-none" />
      <div className="absolute bottom-[-10%] right-[-10%] w-[50%] h-[50%] bg-purple-900/20 rounded-full blur-[120px] pointer-events-none" />

      {/* Grid Pattern overlay */}
      <div className="absolute inset-0 bg-[linear-gradient(to_right,#0f172a_1px,transparent_1px),linear-gradient(to_bottom,#0f172a_1px,transparent_1px)] bg-[size:4rem_4rem] [mask-image:radial-gradient(ellipse_60%_50%_at_50%_50%,#000_70%,transparent_100%)] pointer-events-none opacity-40" />

      {/* Header */}
      <header className="relative max-w-7xl mx-auto px-6 h-20 flex items-center justify-between border-b border-slate-900 z-10">
        <div className="flex items-center gap-3">
          <div className="p-2 bg-indigo-600/10 rounded-xl border border-indigo-500/20 text-indigo-400">
            <KanbanSquare className="w-6 h-6" />
          </div>
          <span className="text-xl font-bold tracking-tight bg-gradient-to-r from-indigo-200 via-purple-100 to-indigo-200 bg-clip-text text-transparent">
            FlowBoard
          </span>
        </div>

        <div className="flex items-center gap-4">
          {isAuthenticated ? (
            <Link
              href="/dashboard"
              className="px-4 py-2 text-sm font-medium bg-slate-900 border border-slate-800 rounded-xl hover:bg-slate-850 hover:border-slate-700 transition"
            >
              Go to Dashboard
            </Link>
          ) : (
            <>
              <button
                onClick={login}
                className="px-4 py-2 text-sm font-medium text-slate-300 hover:text-white transition"
              >
                Sign In
              </button>
              <button
                onClick={login}
                className="px-4 py-2 text-sm font-medium bg-indigo-600 hover:bg-indigo-500 text-white rounded-xl shadow-lg shadow-indigo-600/25 transition flex items-center gap-1.5"
              >
                Get Started <ArrowRight className="w-4 h-4" />
              </button>
            </>
          )}
        </div>
      </header>

      {/* Hero Section */}
      <main className="relative max-w-7xl mx-auto px-6 pt-20 pb-32 z-10 flex flex-col items-center text-center">
        <div className="inline-flex items-center gap-2 px-3.5 py-1.5 rounded-full border border-indigo-500/20 bg-indigo-950/20 text-xs font-semibold text-indigo-300 mb-8 backdrop-blur-md">
          <Sparkles className="w-3.5 h-3.5" /> Next-Gen AI Productivity OS
        </div>

        <h1 className="text-5xl md:text-7xl font-extrabold tracking-tight mb-6 max-w-4xl bg-gradient-to-b from-white via-slate-100 to-slate-400 bg-clip-text text-transparent">
          The collaborative workspace powered by Gemini AI.
        </h1>

        <p className="text-lg md:text-xl text-slate-400 max-w-2xl mb-12 font-light">
          FlowBoard brings together your board pipelines, timelines, sprints, calendars, and real-time team collaboration in one single premium platform.
        </p>

        <div className="flex flex-col sm:flex-row items-center gap-4 mb-20">
          <button
            onClick={login}
            className="w-full sm:w-auto px-8 py-4 bg-indigo-600 hover:bg-indigo-500 text-white font-medium rounded-2xl shadow-xl shadow-indigo-600/20 hover:shadow-indigo-600/30 transition duration-200 flex items-center justify-center gap-2 text-base"
          >
            Start Free with Google <ArrowRight className="w-5 h-5" />
          </button>
          <a
            href="#features"
            className="w-full sm:w-auto px-8 py-4 bg-slate-900 hover:bg-slate-850 text-slate-300 hover:text-white font-medium rounded-2xl border border-slate-800 hover:border-slate-700 transition duration-200 text-center"
          >
            Explore Features
          </a>
        </div>

        {/* Feature Grid */}
        <section id="features" className="w-full pt-16 border-t border-slate-900/60">
          <div className="text-center mb-16">
            <h2 className="text-3xl font-bold mb-4 bg-gradient-to-r from-slate-100 to-slate-300 bg-clip-text text-transparent">
              Engineered for absolute productivity.
            </h2>
            <p className="text-slate-400 font-light">
              Everything you need to ship products, coordinate sprints, and review workloads.
            </p>
          </div>

          <div className="grid md:grid-cols-2 lg:grid-cols-3 gap-6">
            {features.map((f, i) => (
              <div
                key={i}
                className="p-8 rounded-2xl border border-slate-900 bg-slate-950/40 backdrop-blur-md hover:border-indigo-500/20 hover:bg-slate-900/20 transition-all duration-300 text-left group"
              >
                <div className="mb-5 p-3 w-fit rounded-xl bg-slate-900 border border-slate-800 group-hover:border-indigo-500/20 transition duration-300">
                  {f.icon}
                </div>
                <h3 className="text-lg font-semibold text-slate-200 mb-2">{f.title}</h3>
                <p className="text-sm text-slate-400 font-light leading-relaxed">{f.desc}</p>
              </div>
            ))}
          </div>
        </section>
      </main>

      {/* Footer */}
      <footer className="relative max-w-7xl mx-auto px-6 py-12 border-t border-slate-900 flex flex-col md:flex-row items-center justify-between text-sm text-slate-500 z-10 gap-4">
        <span>© 2026 FlowBoard. Pair programmed with Antigravity.</span>
        <div className="flex gap-6">
          <a href="#" className="hover:text-slate-300 transition">Terms</a>
          <a href="#" className="hover:text-slate-300 transition">Privacy</a>
          <a href="#" className="hover:text-slate-300 transition">Contact</a>
        </div>
      </footer>
    </div>
  );
}
