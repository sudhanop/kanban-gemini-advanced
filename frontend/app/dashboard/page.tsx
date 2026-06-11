"use client";

import { useEffect, useState } from "react";
import { useAuth } from "@/components/providers/AuthProvider";
import { workspaceApi, Workspace, reminderApi, Reminder, generalApi, Notification } from "@/lib/api";
import {
  FolderKanban,
  Plus,
  Settings,
  LogOut,
  Bell,
  Clock,
  Sparkles,
  Calendar,
  Video,
  ArrowRight,
  TrendingUp,
  Search,
} from "lucide-react";
import Link from "next/link";
import toast from "react-hot-toast";

export default function DashboardPage() {
  const { user, logout, loading: authLoading, isAuthenticated } = useAuth();
  const [workspaces, setWorkspaces] = useState<Workspace[]>([]);
  const [reminders, setReminders] = useState<Reminder[]>([]);
  const [notifications, setNotifications] = useState<Notification[]>([]);
  const [loading, setLoading] = useState(true);

  // Modal State for new workspace
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [newWsName, setNewWsName] = useState("");
  const [newWsType, setNewWsType] = useState("team");
  const [newWsColor, setNewWsColor] = useState("#6366f1");

  useEffect(() => {
    async function fetchData() {
      try {
        const [wsData, reminderData, notifData] = await Promise.all([
          workspaceApi.list(),
          reminderApi.list(),
          generalApi.notifications(),
        ]);
        setWorkspaces(wsData);
        setReminders(reminderData);
        setNotifications(notifData);
      } catch (err) {
        console.error("Failed to load dashboard data", err);
        toast.error("Error loading dashboard details");
      } finally {
        setLoading(false);
      }
    }
    if (authLoading) return;
    if (!isAuthenticated) return;
    fetchData();
  }, [authLoading, isAuthenticated]);

  const handleCreateWorkspace = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newWsName.trim()) {
      toast.error("Workspace name is required");
      return;
    }
    try {
      const newWs = await workspaceApi.create({
        name: newWsName,
        type: newWsType,
        color: newWsColor,
      });
      setWorkspaces((prev) => [...prev, newWs]);
      setIsModalOpen(false);
      setNewWsName("");
      toast.success("Workspace created successfully!");
    } catch (err) {
      console.error("Create workspace failed", err);
      toast.error("Failed to create workspace");
    }
  };

  const handleToggleReminder = async (rem: Reminder) => {
    try {
      if (rem.status === "active") {
        await reminderApi.pause(rem.id);
        toast.success("Reminder paused");
      } else {
        await reminderApi.resume(rem.id);
        toast.success("Reminder resumed");
      }
      // Reload reminders
      const reminderData = await reminderApi.list();
      setReminders(reminderData);
    } catch (err) {
      toast.error("Failed to update reminder status");
    }
  };

  const handleMarkRead = async (id: string) => {
    try {
      await generalApi.markNotificationRead(id);
      setNotifications((prev) =>
        prev.map((n) => (n.id === id ? { ...n, is_read: true } : n))
      );
    } catch (err) {
      console.error(err);
    }
  };

  if (authLoading || loading) {
    return (
      <div className="min-h-screen bg-slate-950 text-slate-100 flex flex-col items-center justify-center font-sans">
        <div className="w-10 h-10 border-4 border-indigo-500/20 border-t-indigo-500 rounded-full animate-spin mb-4" />
        <p className="text-sm text-slate-400 font-light">Loading workspace dashboard...</p>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-slate-950 text-slate-100 flex font-sans">
      {/* Sidebar */}
      <aside className="w-72 border-r border-slate-900 bg-slate-950/80 backdrop-blur-md flex flex-col shrink-0">
        <div className="h-20 px-6 border-b border-slate-900 flex items-center gap-3">
          <div className="p-2 bg-indigo-600/10 rounded-xl border border-indigo-500/20 text-indigo-400">
            <FolderKanban className="w-5 h-5" />
          </div>
          <span className="font-bold text-lg bg-gradient-to-r from-white to-slate-300 bg-clip-text text-transparent">
            FlowBoard
          </span>
        </div>

        {/* User Card */}
        <div className="p-6 border-b border-slate-900 flex items-center gap-3">
          <img
            src={user?.avatar || "https://images.unsplash.com/photo-1535713875002-d1d0cf377fde"}
            alt="avatar"
            className="w-10 h-10 rounded-xl object-cover ring-1 ring-slate-800"
          />
          <div className="overflow-hidden">
            <div className="font-medium text-sm text-white truncate">{user?.name}</div>
            <div className="text-xs text-slate-500 truncate">{user?.email}</div>
          </div>
        </div>

        {/* Navigation / Workspace List */}
        <div className="flex-1 px-4 py-6 overflow-y-auto space-y-6">
          <div>
            <div className="flex items-center justify-between px-2 text-xs font-semibold text-slate-500 uppercase tracking-wider mb-3">
              <span>Workspaces</span>
              <button
                onClick={() => setIsModalOpen(true)}
                className="text-indigo-400 hover:text-indigo-300 transition"
              >
                <Plus className="w-4 h-4" />
              </button>
            </div>
            <div className="space-y-1">
              {workspaces.map((ws) => (
                <Link
                  key={ws.id}
                  href={`/workspaces/${ws.id}`}
                  className="flex items-center justify-between px-3 py-2 text-sm rounded-xl text-slate-400 hover:text-white hover:bg-slate-900 border border-transparent hover:border-slate-850 transition duration-150"
                >
                  <div className="flex items-center gap-2.5 overflow-hidden">
                    <div
                      className="w-3 h-3 rounded-full shrink-0"
                      style={{ backgroundColor: ws.color }}
                    />
                    <span className="truncate">{ws.name}</span>
                  </div>
                  <ArrowRight className="w-4 h-4 text-slate-600 group-hover:text-slate-400 transition" />
                </Link>
              ))}
            </div>
          </div>
        </div>

        {/* Footer actions */}
        <div className="p-4 border-t border-slate-900 space-y-1">
          <button
            onClick={() => toast.success("Settings available in individual workspaces")}
            className="w-full flex items-center gap-3 px-3 py-2 text-sm text-slate-400 hover:text-white rounded-xl hover:bg-slate-900 transition"
          >
            <Settings className="w-4 h-4" /> Settings
          </button>
          <button
            onClick={logout}
            className="w-full flex items-center gap-3 px-3 py-2 text-sm text-rose-400 hover:text-rose-300 rounded-xl hover:bg-rose-500/10 transition"
          >
            <LogOut className="w-4 h-4" /> Logout
          </button>
        </div>
      </aside>

      {/* Main Dashboard Area */}
      <main className="flex-1 overflow-y-auto p-8 md:p-12 relative">
        <header className="flex items-center justify-between mb-10 pb-6 border-b border-slate-900">
          <div>
            <h1 className="text-3xl font-extrabold tracking-tight bg-gradient-to-r from-white via-slate-100 to-slate-400 bg-clip-text text-transparent">
              Dashboard
            </h1>
            <p className="text-slate-500 text-sm font-light">Welcome back, {user?.name}</p>
          </div>
          <div className="flex items-center gap-3">
            <div className="px-3.5 py-1.5 rounded-full border border-indigo-500/20 bg-indigo-950/20 text-xs font-semibold text-indigo-300 flex items-center gap-1.5 backdrop-blur-md">
              <Sparkles className="w-3.5 h-3.5 animate-pulse" /> AI Assistant Online
            </div>
          </div>
        </header>

        {/* Content grid */}
        <div className="grid lg:grid-cols-3 gap-8">
          {/* Left / Middle: Workspaces grid & summary */}
          <div className="lg:col-span-2 space-y-8">
            <section>
              <div className="flex items-center justify-between mb-4">
                <h2 className="text-lg font-semibold text-slate-200">Active Workspaces</h2>
                <button
                  onClick={() => setIsModalOpen(true)}
                  className="px-3 py-1.5 text-xs font-medium bg-indigo-600 hover:bg-indigo-500 text-white rounded-xl shadow-md transition duration-150 flex items-center gap-1"
                >
                  <Plus className="w-3.5 h-3.5" /> New Workspace
                </button>
              </div>

              {workspaces.length === 0 ? (
                <div className="p-12 border border-dashed border-slate-800 rounded-2xl text-center">
                  <FolderKanban className="w-10 h-10 text-slate-600 mx-auto mb-4" />
                  <p className="text-sm text-slate-400 font-light mb-4">No workspaces found. Create your first workspace to start managing tasks.</p>
                  <button
                    onClick={() => setIsModalOpen(true)}
                    className="px-4 py-2 text-sm font-medium bg-indigo-600 hover:bg-indigo-500 text-white rounded-xl shadow-md transition duration-150"
                  >
                    Create Workspace
                  </button>
                </div>
              ) : (
                <div className="grid md:grid-cols-2 gap-4">
                  {workspaces.map((ws) => (
                    <Link
                      key={ws.id}
                      href={`/workspaces/${ws.id}`}
                      className="group p-6 rounded-2xl border border-slate-900 bg-slate-950/40 backdrop-blur-md hover:border-slate-800 hover:bg-slate-900/40 transition duration-200 text-left block"
                    >
                      <div className="flex justify-between items-start mb-4">
                        <div
                          className="w-4 h-4 rounded-full"
                          style={{ backgroundColor: ws.color }}
                        />
                        <span className="text-[10px] uppercase tracking-wider font-semibold text-indigo-400 px-2 py-0.5 bg-indigo-600/10 border border-indigo-500/10 rounded-md">
                          {ws.type}
                        </span>
                      </div>
                      <h3 className="font-semibold text-slate-200 group-hover:text-white transition mb-1 truncate">
                        {ws.name}
                      </h3>
                      <p className="text-xs text-slate-500 font-light truncate mb-4">
                        slug: /{ws.slug}
                      </p>
                      <div className="flex items-center justify-between text-xs text-slate-400 pt-4 border-t border-slate-900/60 group-hover:border-slate-800 transition">
                        <span className="font-light">Manage Board & Timeline</span>
                        <ArrowRight className="w-4 h-4 text-slate-600 group-hover:text-indigo-400 group-hover:translate-x-1 transition duration-200" />
                      </div>
                    </Link>
                  ))}
                </div>
              )}
            </section>
          </div>

          {/* Right Column: Reminders & Notifications */}
          <div className="space-y-8">
            {/* Active Reminders */}
            <section className="p-6 rounded-2xl border border-slate-900 bg-slate-950/40 backdrop-blur-md">
              <h2 className="text-sm font-semibold text-slate-300 flex items-center gap-2 mb-4">
                <Clock className="w-4 h-4 text-indigo-400" /> Reminders
              </h2>

              {reminders.length === 0 ? (
                <p className="text-xs text-slate-500 font-light py-4 text-center">No active reminders.</p>
              ) : (
                <div className="space-y-3">
                  {reminders.map((rem) => (
                    <div
                      key={rem.id}
                      className="p-3 rounded-xl border border-slate-900 bg-slate-900/20 flex items-center justify-between gap-3 text-xs"
                    >
                      <div className="overflow-hidden">
                        <p className="text-slate-300 font-medium truncate">{rem.message}</p>
                        <p className="text-slate-500 font-light text-[10px] mt-0.5">
                          {new Date(rem.remind_at).toLocaleDateString()} at{" "}
                          {new Date(rem.remind_at).toLocaleTimeString([], {
                            hour: "2-digit",
                            minute: "2-digit",
                          })}
                        </p>
                      </div>
                      <button
                        onClick={() => handleToggleReminder(rem)}
                        className={`px-2 py-1 rounded-md text-[10px] font-medium border transition ${
                          rem.status === "active"
                            ? "bg-slate-900 border-slate-800 text-slate-400 hover:text-white"
                            : "bg-indigo-600/10 border-indigo-500/20 text-indigo-400 hover:bg-indigo-600 hover:text-white"
                        }`}
                      >
                        {rem.status === "active" ? "Pause" : "Resume"}
                      </button>
                    </div>
                  ))}
                </div>
              )}
            </section>

            {/* Notifications */}
            <section className="p-6 rounded-2xl border border-slate-900 bg-slate-950/40 backdrop-blur-md">
              <h2 className="text-sm font-semibold text-slate-300 flex items-center gap-2 mb-4">
                <Bell className="w-4 h-4 text-purple-400" /> Notifications
              </h2>

              {notifications.length === 0 ? (
                <p className="text-xs text-slate-500 font-light py-4 text-center">No notifications.</p>
              ) : (
                <div className="space-y-3">
                  {notifications.map((notif) => (
                    <div
                      key={notif.id}
                      onClick={() => !notif.is_read && handleMarkRead(notif.id)}
                      className={`p-3 rounded-xl border transition text-xs cursor-pointer ${
                        notif.is_read
                          ? "bg-slate-950/20 border-slate-900/60 text-slate-500"
                          : "bg-indigo-600/5 border-indigo-500/10 text-slate-300 hover:bg-indigo-600/10"
                      }`}
                    >
                      <p className="font-semibold mb-0.5">{notif.title}</p>
                      <p className="font-light text-slate-400 mb-1">{notif.message}</p>
                      <span className="text-[10px] text-slate-600">
                        {new Date(notif.created_at).toLocaleDateString()}
                      </span>
                    </div>
                  ))}
                </div>
              )}
            </section>
          </div>
        </div>
      </main>

      {/* Create Workspace Modal */}
      {isModalOpen && (
        <div className="fixed inset-0 bg-black/60 backdrop-blur-sm flex items-center justify-center z-50 p-4">
          <div className="w-full max-w-md p-6 rounded-2xl border border-slate-800 bg-slate-950 shadow-2xl flex flex-col">
            <h3 className="text-lg font-bold text-white mb-2">Create Workspace</h3>
            <p className="text-xs text-slate-500 font-light mb-6">
              Create a project or team workspace to group your Kanban boards.
            </p>

            <form onSubmit={handleCreateWorkspace} className="space-y-4">
              <div>
                <label className="block text-xs text-slate-400 font-medium mb-1.5">Workspace Name</label>
                <input
                  type="text"
                  value={newWsName}
                  onChange={(e) => setNewWsName(e.target.value)}
                  placeholder="e.g. Acme Project, Dev Sprint"
                  className="w-full px-3.5 py-2.5 rounded-xl border border-slate-800 bg-slate-900 text-sm text-white focus:outline-none focus:border-indigo-500 transition placeholder-slate-600"
                />
              </div>

              <div>
                <label className="block text-xs text-slate-400 font-medium mb-1.5">Workspace Type</label>
                <select
                  value={newWsType}
                  onChange={(e) => setNewWsType(e.target.value)}
                  className="w-full px-3.5 py-2.5 rounded-xl border border-slate-800 bg-slate-900 text-sm text-white focus:outline-none focus:border-indigo-500 transition"
                >
                  <option value="personal">Personal Project</option>
                  <option value="team">Team Project</option>
                  <option value="enterprise">Enterprise Portfolio</option>
                </select>
              </div>

              <div>
                <label className="block text-xs text-slate-400 font-medium mb-1.5">Accent Color</label>
                <div className="flex gap-2.5 pt-1">
                  {["#6366f1", "#a855f7", "#ec4899", "#10b981", "#f59e0b", "#0ea5e9"].map((color) => (
                    <button
                      key={color}
                      type="button"
                      onClick={() => setNewWsColor(color)}
                      className={`w-7 h-7 rounded-full border-2 transition ${
                        newWsColor === color ? "border-white" : "border-transparent"
                      }`}
                      style={{ backgroundColor: color }}
                    />
                  ))}
                </div>
              </div>

              <div className="flex items-center justify-end gap-3 pt-6">
                <button
                  type="button"
                  onClick={() => setIsModalOpen(false)}
                  className="px-4 py-2 rounded-xl text-sm border border-slate-850 hover:bg-slate-900 text-slate-400 hover:text-white transition"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  className="px-4 py-2 rounded-xl text-sm bg-indigo-600 hover:bg-indigo-500 text-white font-medium shadow-lg transition"
                >
                  Create
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
