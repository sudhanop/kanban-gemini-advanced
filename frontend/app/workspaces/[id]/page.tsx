"use client";

import { useEffect, useState } from "react";
import { useAuth } from "@/components/providers/AuthProvider";
import {
  workspaceApi,
  boardApi,
  generalApi,
  Workspace,
  Board,
  WorkspaceMember,
  Meeting,
} from "@/lib/api";
import {
  FolderKanban,
  Plus,
  Users,
  Video,
  BarChart2,
  Settings,
  ArrowRight,
  TrendingUp,
  Mail,
  UserPlus,
  X,
  Trash2,
} from "lucide-react";
import Link from "next/link";
import toast from "react-hot-toast";
import { useParams, useRouter } from "next/navigation";

export default function WorkspacePage() {
  const { id } = useParams() as { id: string };
  const { user, loading: authLoading, isAuthenticated } = useAuth();
  const router = useRouter();

  const [workspace, setWorkspace] = useState<Workspace | null>(null);
  const [boards, setBoards] = useState<Board[]>([]);
  const [members, setMembers] = useState<WorkspaceMember[]>([]);
  const [meetings, setMeetings] = useState<Meeting[]>([]);
  const [stats, setStats] = useState<any>(null);
  const [loading, setLoading] = useState(true);

  // Modals state
  const [isBoardModalOpen, setIsBoardModalOpen] = useState(false);
  const [isInviteModalOpen, setIsInviteModalOpen] = useState(false);
  const [isMeetingModalOpen, setIsMeetingModalOpen] = useState(false);
  const [isSettingsModalOpen, setIsSettingsModalOpen] = useState(false);

  // Form states
  const [boardName, setBoardName] = useState("");
  const [boardDesc, setBoardDesc] = useState("");
  const [boardColor, setBoardColor] = useState("#6366f1");

  const [inviteEmail, setInviteEmail] = useState("");
  const [inviteRole, setInviteRole] = useState("member");

  const [meetingTitle, setMeetingTitle] = useState("");
  const [meetingDesc, setMeetingDesc] = useState("");
  const [meetingTime, setMeetingTime] = useState("");

  const [editWsName, setEditWsName] = useState("");
  const [editWsType, setEditWsType] = useState("team");
  const [editWsColor, setEditWsColor] = useState("#6366f1");

  const fetchData = async () => {
    try {
      const [ws, bds, mems, st, mts] = await Promise.all([
        workspaceApi.get(id),
        boardApi.list(id),
        workspaceApi.getMembers(id),
        generalApi.getAnalytics(id),
        generalApi.listMeetings(id),
      ]);
      setWorkspace(ws);
      setBoards(bds);
      setMembers(mems);
      setStats(st);
      setMeetings(mts);
      
      setEditWsName(ws.name);
      setEditWsType(ws.type);
      setEditWsColor(ws.color);
    } catch (err) {
      console.error(err);
      toast.error("Failed to load workspace data");
      router.push("/dashboard");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (!id) return;
    if (authLoading) return;
    if (!isAuthenticated) return;
    fetchData();
  }, [id, authLoading, isAuthenticated]);

  const handleCreateBoard = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!boardName.trim()) return;
    try {
      const newBoard = await boardApi.create(id, {
        name: boardName,
        description: boardDesc,
        color: boardColor,
      });
      setBoards((prev) => [...prev, newBoard]);
      setIsBoardModalOpen(false);
      setBoardName("");
      setBoardDesc("");
      toast.success("Board created successfully!");
    } catch (err) {
      toast.error("Failed to create board");
    }
  };

  const handleInviteUser = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!inviteEmail.trim()) return;
    try {
      await workspaceApi.inviteUser(id, inviteEmail, inviteRole);
      setIsInviteModalOpen(false);
      setInviteEmail("");
      toast.success("Invitation email sent!");
    } catch (err) {
      toast.error("Failed to send invitation");
    }
  };

  const handleCreateMeeting = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!meetingTitle.trim() || !meetingTime) return;
    try {
      const newMeeting = await generalApi.createMeeting({
        title: meetingTitle,
        description: meetingDesc,
        scheduled_at: new Date(meetingTime).toISOString(),
        workspace_id: id,
      });
      setMeetings((prev) => [...prev, newMeeting]);
      setIsMeetingModalOpen(false);
      setMeetingTitle("");
      setMeetingDesc("");
      setMeetingTime("");
      toast.success("Meeting scheduled successfully!");
    } catch (err) {
      toast.error("Failed to schedule meeting");
    }
  };
  const handleUpdateWorkspace = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!editWsName.trim()) return;
    try {
      const updated = await workspaceApi.update(id, {
        name: editWsName,
        type: editWsType,
        color: editWsColor,
      });
      setWorkspace(updated);
      setIsSettingsModalOpen(false);
      toast.success("Workspace updated successfully!");
    } catch (err) {
      toast.error("Failed to update workspace");
    }
  };

  const handleDeleteWorkspace = async () => {
    if (!confirm("Are you sure you want to permanently delete this workspace and all its boards? This action cannot be undone.")) return;
    try {
      await workspaceApi.delete(id);
      toast.success("Workspace deleted");
      router.push("/dashboard");
    } catch (err) {
      toast.error("Failed to delete workspace");
    }
  };

  const handleRemoveMember = async (memberId: string) => {
    if (!confirm("Remove this member from the workspace?")) return;
    try {
      await workspaceApi.removeMember(id, memberId);
      setMembers((prev) => prev.filter((m) => m.id !== memberId));
      toast.success("Member removed");
    } catch (err) {
      toast.error("Failed to remove member");
    }
  };

  if (authLoading || loading || !workspace) {
    return (
      <div className="min-h-screen bg-slate-950 text-slate-100 flex flex-col items-center justify-center font-sans">
        <div className="w-10 h-10 border-4 border-indigo-500/20 border-t-indigo-500 rounded-full animate-spin mb-4" />
        <p className="text-sm text-slate-400 font-light">Loading workspace detail...</p>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-slate-950 text-slate-100 font-sans flex flex-col">
      {/* Navbar header */}
      <header className="h-20 border-b border-slate-900 bg-slate-950/80 backdrop-blur-md px-8 md:px-12 flex items-center justify-between sticky top-0 z-40">
        <div className="flex items-center gap-4">
          <Link href="/dashboard" className="text-sm font-medium text-slate-400 hover:text-white transition">
            ← Dashboard
          </Link>
          <span className="text-slate-700">/</span>
          <div className="flex items-center gap-2">
            <div className="w-3.5 h-3.5 rounded-full" style={{ backgroundColor: workspace.color }} />
            <h1 className="font-semibold text-white text-base">{workspace.name}</h1>
          </div>
        </div>

        <div className="flex items-center gap-3">
          <button
            onClick={() => setIsInviteModalOpen(true)}
            className="px-3.5 py-1.5 rounded-xl border border-slate-850 hover:bg-slate-900 text-xs font-medium text-slate-300 hover:text-white transition flex items-center gap-1.5"
          >
            <UserPlus className="w-4 h-4" /> Invite Member
          </button>
          <button
            onClick={() => setIsMeetingModalOpen(true)}
            className="px-3.5 py-1.5 rounded-xl border border-indigo-500/20 bg-indigo-600/10 text-xs font-medium text-indigo-400 hover:bg-indigo-650 hover:text-white transition flex items-center gap-1.5"
          >
            <Video className="w-4 h-4" /> Schedule Standup
          </button>
          <button
            onClick={() => setIsSettingsModalOpen(true)}
            className="px-3.5 py-1.5 rounded-xl border border-slate-850 hover:bg-slate-900 text-xs font-medium text-slate-300 hover:text-white transition flex items-center gap-1.5"
          >
            <Settings className="w-4 h-4" /> Settings
          </button>
        </div>
      </header>

      {/* Main Container */}
      <main className="max-w-7xl mx-auto w-full px-8 md:px-12 py-10 flex-1 grid lg:grid-cols-3 gap-10">
        {/* Left/Center columns - Boards & Analytics */}
        <div className="lg:col-span-2 space-y-10">
          {/* Boards Section */}
          <section>
            <div className="flex items-center justify-between mb-6">
              <h2 className="text-xl font-bold text-white flex items-center gap-2">
                <FolderKanban className="w-5 h-5 text-indigo-400" /> Boards
              </h2>
              <button
                onClick={() => setIsBoardModalOpen(true)}
                className="px-3 py-1.5 text-xs font-medium bg-indigo-600 hover:bg-indigo-500 text-white rounded-xl shadow-md transition flex items-center gap-1"
              >
                <Plus className="w-3.5 h-3.5" /> Create Board
              </button>
            </div>

            {boards.length === 0 ? (
              <div className="p-12 border border-dashed border-slate-850 bg-slate-950/20 backdrop-blur-md rounded-2xl text-center">
                <FolderKanban className="w-10 h-10 text-slate-700 mx-auto mb-4" />
                <p className="text-sm text-slate-400 font-light mb-4">No boards created yet.</p>
                <button
                  onClick={() => setIsBoardModalOpen(true)}
                  className="px-4 py-2 text-sm font-medium bg-indigo-600 hover:bg-indigo-500 text-white rounded-xl shadow-md transition"
                >
                  Create Board
                </button>
              </div>
            ) : (
              <div className="grid sm:grid-cols-2 gap-4">
                {boards.map((b) => (
                  <Link
                    key={b.id}
                    href={`/boards/${b.id}?ws=${workspace.id}`}
                    className="group p-6 rounded-2xl border border-slate-900 bg-slate-950/40 backdrop-blur-md hover:border-slate-800 hover:bg-slate-900/40 transition duration-200 block"
                  >
                    <div className="flex justify-between items-start mb-4">
                      <div className="w-3.5 h-3.5 rounded-full" style={{ backgroundColor: b.color }} />
                      <span className="text-[10px] uppercase font-semibold text-slate-500">
                        {b.view_type || "Kanban"}
                      </span>
                    </div>
                    <h3 className="font-semibold text-slate-200 group-hover:text-white transition mb-1">
                      {b.name}
                    </h3>
                    <p className="text-xs text-slate-500 font-light line-clamp-2 leading-relaxed">
                      {b.description || "No description provided."}
                    </p>
                    <div className="flex items-center justify-between text-xs text-slate-400 pt-4 mt-4 border-t border-slate-900/60 group-hover:border-slate-800 transition">
                      <span className="font-light">Open board view</span>
                      <ArrowRight className="w-4 h-4 text-slate-600 group-hover:text-indigo-400 group-hover:translate-x-1 transition duration-200" />
                    </div>
                  </Link>
                ))}
              </div>
            )}
          </section>

          {/* Quick Analytics Summary */}
          {stats && (
            <section className="space-y-4">
              <h2 className="text-lg font-semibold text-slate-200 flex items-center gap-2">
                <BarChart2 className="w-5 h-5 text-purple-400" /> Workspace Insights
              </h2>
              <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                {[
                  { label: "Total Boards", value: stats.total_boards || boards.length },
                  { label: "Total Tasks", value: stats.total_tasks || 0 },
                  { label: "In Progress", value: stats.in_progress_tasks || 0 },
                  { label: "Completed", value: stats.completed_tasks || 0 },
                ].map((s, i) => (
                  <div key={i} className="p-5 rounded-2xl border border-slate-900 bg-slate-950/20 text-center">
                    <div className="text-2xl font-extrabold text-white mb-1">{s.value}</div>
                    <div className="text-[11px] text-slate-500 font-medium uppercase tracking-wider">{s.label}</div>
                  </div>
                ))}
              </div>
            </section>
          )}
        </div>

        {/* Right column - Members & Meetings details */}
        <div className="space-y-8">
          {/* Members */}
          <section className="p-6 rounded-2xl border border-slate-900 bg-slate-950/40 backdrop-blur-md">
            <h2 className="text-sm font-semibold text-slate-300 flex items-center gap-2 mb-4">
              <Users className="w-4 h-4 text-emerald-400" /> Team Members ({members.length})
            </h2>

            <div className="space-y-4">
              {members.map((m) => (
                <div key={m.id} className="flex items-center justify-between gap-3">
                  <div className="flex items-center gap-3">
                    <img
                      src={m.user?.avatar || "https://images.unsplash.com/photo-1535713875002-d1d0cf377fde"}
                      alt="member"
                      className="w-8 h-8 rounded-lg object-cover ring-1 ring-slate-800"
                    />
                    <div className="overflow-hidden">
                      <p className="text-xs font-semibold text-slate-200 truncate">{m.user?.name}</p>
                      <p className="text-[10px] text-slate-500 truncate">{m.user?.email}</p>
                    </div>
                  </div>
                  <span className="text-[10px] uppercase font-semibold text-slate-500 px-2 py-0.5 border border-slate-800 rounded-md">
                    {m.role}
                  </span>
                </div>
              ))}
            </div>
          </section>

          {/* Video Meetings list */}
          <section className="p-6 rounded-2xl border border-slate-900 bg-slate-950/40 backdrop-blur-md">
            <h2 className="text-sm font-semibold text-slate-300 flex items-center gap-2 mb-4">
              <Video className="w-4 h-4 text-sky-400" /> Scheduled Video Standups
            </h2>

            {meetings.length === 0 ? (
              <p className="text-xs text-slate-500 font-light py-4 text-center">No active workspace standups.</p>
            ) : (
              <div className="space-y-3">
                {meetings.map((m) => (
                  <div key={m.id} className="p-3.5 rounded-xl border border-slate-900 bg-slate-900/20 text-xs">
                    <div className="flex justify-between items-start mb-2">
                      <p className="font-semibold text-slate-200">{m.title}</p>
                      <span className="text-[10px] uppercase font-semibold text-indigo-400 px-1.5 py-0.5 bg-indigo-600/10 rounded-md">
                        {m.meeting_type || "jitsi"}
                      </span>
                    </div>
                    <p className="text-slate-400 font-light mb-3">{m.description}</p>
                    <a
                      href={m.meeting_link}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-indigo-650 hover:bg-indigo-600 text-white font-medium text-[11px] transition shadow-md"
                    >
                      Join Meeting
                    </a>
                  </div>
                ))}
              </div>
            )}
          </section>
        </div>
      </main>

      {/* CREATE BOARD MODAL */}
      {isBoardModalOpen && (
        <div className="fixed inset-0 bg-black/60 backdrop-blur-sm flex items-center justify-center z-50 p-4">
          <div className="w-full max-w-md p-6 rounded-2xl border border-slate-800 bg-slate-950 shadow-2xl flex flex-col">
            <h3 className="text-lg font-bold text-white mb-2">Create Kanban Board</h3>
            <p className="text-xs text-slate-500 font-light mb-6">Create a board to manage tasks, sprints, and timelines.</p>

            <form onSubmit={handleCreateBoard} className="space-y-4">
              <div>
                <label className="block text-xs text-slate-400 font-medium mb-1.5">Board Title</label>
                <input
                  type="text"
                  value={boardName}
                  onChange={(e) => setBoardName(e.target.value)}
                  placeholder="e.g. Sprint Board, Product Launch"
                  className="w-full px-3.5 py-2.5 rounded-xl border border-slate-800 bg-slate-900 text-sm text-white focus:outline-none focus:border-indigo-500 transition placeholder-slate-600"
                />
              </div>

              <div>
                <label className="block text-xs text-slate-400 font-medium mb-1.5">Description (optional)</label>
                <textarea
                  value={boardDesc}
                  onChange={(e) => setBoardDesc(e.target.value)}
                  placeholder="Summarize board goal..."
                  className="w-full h-20 px-3.5 py-2.5 rounded-xl border border-slate-800 bg-slate-900 text-sm text-white focus:outline-none focus:border-indigo-500 transition resize-none placeholder-slate-600"
                />
              </div>

              <div>
                <label className="block text-xs text-slate-400 font-medium mb-1.5">Label Color</label>
                <div className="flex gap-2.5 pt-1">
                  {["#6366f1", "#a855f7", "#ec4899", "#10b981", "#f59e0b", "#0ea5e9"].map((color) => (
                    <button
                      key={color}
                      type="button"
                      onClick={() => setBoardColor(color)}
                      className={`w-7 h-7 rounded-full border-2 transition ${
                        boardColor === color ? "border-white" : "border-transparent"
                      }`}
                      style={{ backgroundColor: color }}
                    />
                  ))}
                </div>
              </div>

              <div className="flex items-center justify-end gap-3 pt-6">
                <button
                  type="button"
                  onClick={() => setIsBoardModalOpen(false)}
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

      {/* INVITE MEMBER MODAL */}
      {isInviteModalOpen && (
        <div className="fixed inset-0 bg-black/60 backdrop-blur-sm flex items-center justify-center z-50 p-4">
          <div className="w-full max-w-md p-6 rounded-2xl border border-slate-800 bg-slate-950 shadow-2xl flex flex-col">
            <h3 className="text-lg font-bold text-white mb-2">Invite Collaborator</h3>
            <p className="text-xs text-slate-500 font-light mb-6">Send an email invitation link to join this workspace.</p>

            <form onSubmit={handleInviteUser} className="space-y-4">
              <div>
                <label className="block text-xs text-slate-400 font-medium mb-1.5">Email Address</label>
                <input
                  type="email"
                  value={inviteEmail}
                  onChange={(e) => setInviteEmail(e.target.value)}
                  placeholder="colleague@company.com"
                  className="w-full px-3.5 py-2.5 rounded-xl border border-slate-800 bg-slate-900 text-sm text-white focus:outline-none focus:border-indigo-500 transition placeholder-slate-600"
                />
              </div>

              <div>
                <label className="block text-xs text-slate-400 font-medium mb-1.5">Role Permission</label>
                <select
                  value={inviteRole}
                  onChange={(e) => setInviteRole(e.target.value)}
                  className="w-full px-3.5 py-2.5 rounded-xl border border-slate-800 bg-slate-900 text-sm text-white focus:outline-none focus:border-indigo-500 transition"
                >
                  <option value="admin">Administrator</option>
                  <option value="member">Regular Member</option>
                  <option value="viewer">Read-Only Viewer</option>
                </select>
              </div>

              <div className="flex items-center justify-end gap-3 pt-6">
                <button
                  type="button"
                  onClick={() => setIsInviteModalOpen(false)}
                  className="px-4 py-2 rounded-xl text-sm border border-slate-850 hover:bg-slate-900 text-slate-400 hover:text-white transition"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  className="px-4 py-2 rounded-xl text-sm bg-indigo-600 hover:bg-indigo-500 text-white font-medium shadow-lg transition"
                >
                  Send Invite
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* SCHEDULE STANDUP MODAL */}
      {isMeetingModalOpen && (
        <div className="fixed inset-0 bg-black/60 backdrop-blur-sm flex items-center justify-center z-50 p-4">
          <div className="w-full max-w-md p-6 rounded-2xl border border-slate-800 bg-slate-950 shadow-2xl flex flex-col">
            <h3 className="text-lg font-bold text-white mb-2">Schedule Standup Meeting</h3>
            <p className="text-xs text-slate-500 font-light mb-6">Create an instant Jitsi Meet video standup room for your team.</p>

            <form onSubmit={handleCreateMeeting} className="space-y-4">
              <div>
                <label className="block text-xs text-slate-400 font-medium mb-1.5">Meeting Topic</label>
                <input
                  type="text"
                  value={meetingTitle}
                  onChange={(e) => setMeetingTitle(e.target.value)}
                  placeholder="e.g. Daily Standup, Sprint Retro"
                  className="w-full px-3.5 py-2.5 rounded-xl border border-slate-800 bg-slate-900 text-sm text-white focus:outline-none focus:border-indigo-500 transition placeholder-slate-600"
                />
              </div>

              <div>
                <label className="block text-xs text-slate-400 font-medium mb-1.5">Agenda / Notes</label>
                <textarea
                  value={meetingDesc}
                  onChange={(e) => setMeetingDesc(e.target.value)}
                  placeholder="Meeting agenda notes..."
                  className="w-full h-20 px-3.5 py-2.5 rounded-xl border border-slate-800 bg-slate-900 text-sm text-white focus:outline-none focus:border-indigo-500 transition resize-none placeholder-slate-600"
                />
              </div>

              <div>
                <label className="block text-xs text-slate-400 font-medium mb-1.5">Scheduled Date & Time</label>
                <input
                  type="datetime-local"
                  value={meetingTime}
                  onChange={(e) => setMeetingTime(e.target.value)}
                  className="w-full px-3.5 py-2.5 rounded-xl border border-slate-800 bg-slate-900 text-sm text-white focus:outline-none focus:border-indigo-500 transition"
                />
              </div>

              <div className="flex items-center justify-end gap-3 pt-6">
                <button
                  type="button"
                  onClick={() => setIsMeetingModalOpen(false)}
                  className="px-4 py-2 rounded-xl text-sm border border-slate-850 hover:bg-slate-900 text-slate-400 hover:text-white transition"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  className="px-4 py-2 rounded-xl text-sm bg-indigo-600 hover:bg-indigo-500 text-white font-medium shadow-lg transition"
                >
                  Schedule
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* WORKSPACE SETTINGS MODAL */}
      {isSettingsModalOpen && (
        <div className="fixed inset-0 bg-black/60 backdrop-blur-sm flex items-center justify-center z-50 p-4">
          <div className="w-full max-w-lg p-6 rounded-2xl border border-slate-800 bg-slate-950 shadow-2xl flex flex-col max-h-[90vh] overflow-y-auto">
            <div className="flex items-center justify-between mb-4 pb-2 border-b border-slate-900">
              <h3 className="text-lg font-bold text-white flex items-center gap-2">
                <Settings className="w-5 h-5 text-indigo-400" /> Workspace Settings
              </h3>
              <button onClick={() => setIsSettingsModalOpen(false)} className="text-slate-400 hover:text-white transition">
                <X className="w-5 h-5" />
              </button>
            </div>

            <form onSubmit={handleUpdateWorkspace} className="space-y-4">
              <div>
                <label className="block text-xs text-slate-400 font-medium mb-1.5">Workspace Name</label>
                <input
                  type="text"
                  value={editWsName}
                  onChange={(e) => setEditWsName(e.target.value)}
                  className="w-full px-3.5 py-2.5 rounded-xl border border-slate-800 bg-slate-900 text-sm text-white focus:outline-none focus:border-indigo-500 transition"
                  placeholder="Workspace Name"
                  required
                />
              </div>

              <div>
                <label className="block text-xs text-slate-400 font-medium mb-1.5">Workspace Type</label>
                <select
                  value={editWsType}
                  onChange={(e) => setEditWsType(e.target.value)}
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
                      onClick={() => setEditWsColor(color)}
                      className={`w-7 h-7 rounded-full border-2 transition ${
                        editWsColor === color ? "border-white" : "border-transparent"
                      }`}
                      style={{ backgroundColor: color }}
                    />
                  ))}
                </div>
              </div>

              <div className="flex justify-between items-center pt-4 border-t border-slate-900">
                <button
                  type="button"
                  onClick={handleDeleteWorkspace}
                  className="px-4 py-2 rounded-xl text-sm bg-rose-600/10 border border-rose-500/20 text-rose-400 hover:bg-rose-650 hover:text-white transition"
                >
                  Delete Workspace
                </button>
                <div className="flex gap-3">
                  <button
                    type="button"
                    onClick={() => setIsSettingsModalOpen(false)}
                    className="px-4 py-2 rounded-xl text-sm border border-slate-850 hover:bg-slate-900 text-slate-400 hover:text-white transition"
                  >
                    Cancel
                  </button>
                  <button
                    type="submit"
                    className="px-4 py-2 rounded-xl text-sm bg-indigo-600 hover:bg-indigo-500 text-white font-medium transition"
                  >
                    Save Changes
                  </button>
                </div>
              </div>
            </form>

            <div className="mt-8 pt-6 border-t border-slate-900">
              <h4 className="text-sm font-semibold text-slate-350 mb-4 flex items-center gap-2">
                <Users className="w-4 h-4 text-emerald-400" /> Manage Team Members
              </h4>
              <div className="space-y-3 max-h-60 overflow-y-auto pr-1">
                {members.map((m) => {
                  const isOwner = m.user_id === workspace.owner_id;
                  return (
                    <div key={m.id} className="flex items-center justify-between gap-3 p-2.5 rounded-xl border border-slate-900/60 bg-slate-900/10">
                      <div className="flex items-center gap-3">
                        <img
                          src={m.user?.avatar || "https://images.unsplash.com/photo-1535713875002-d1d0cf377fde"}
                          alt="member"
                          className="w-8 h-8 rounded-lg object-cover ring-1 ring-slate-800"
                        />
                        <div>
                          <p className="text-xs font-semibold text-slate-200">{m.user?.name}</p>
                          <p className="text-[10px] text-slate-500">{m.user?.email}</p>
                        </div>
                      </div>
                      <div className="flex items-center gap-2">
                        <span className="text-[10px] uppercase font-semibold text-slate-500 px-2 py-0.5 border border-slate-800 rounded-md">
                          {isOwner ? "Owner" : m.role}
                        </span>
                        {!isOwner && (
                          <button
                            onClick={() => handleRemoveMember(m.id)}
                            className="p-1.5 text-slate-600 hover:text-rose-400 transition"
                            title="Remove Member"
                          >
                            <Trash2 className="w-3.5 h-3.5" />
                          </button>
                        )}
                      </div>
                    </div>
                  );
                })}
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
