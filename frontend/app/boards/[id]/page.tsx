"use client";

import { useEffect, useState } from "react";
import { useAuth } from "@/components/providers/AuthProvider";
import { useWebSocket } from "@/components/providers/WebSocketProvider";
import { boardApi, taskApi, Board, Column, Task } from "@/lib/api";
import { KanbanBoard } from "@/components/kanban/KanbanBoard";
import { TimelineView } from "@/components/timeline/TimelineView";
import { SprintView } from "@/components/sprint/SprintView";
import { CalendarView } from "@/components/calendar/CalendarView";
import { AnalyticsDashboard } from "@/components/analytics/AnalyticsDashboard";
import { AIAssistant } from "@/components/ai/AIAssistant";
import { TaskModal } from "@/components/task/TaskModal";
import { BoardSettingsModal } from "@/components/board/BoardSettingsModal";
import {
  FolderKanban,
  Calendar,
  Layers,
  BarChart2,
  Sparkles,
  Settings,
  Share2,
  FileSpreadsheet,
  Copy,
  Archive,
  Activity,
  Users,
} from "lucide-react";
import Link from "next/link";
import toast from "react-hot-toast";
import { useParams, useSearchParams, useRouter } from "next/navigation";

export default function BoardPage() {
  const { id: boardId } = useParams() as { id: string };
  const searchParams = useSearchParams();
  const workspaceId = searchParams.get("ws") || "";
  const router = useRouter();
  const { loading: authLoading, isAuthenticated } = useAuth();

  const { isConnected, joinRoom, leaveRoom, registerListener } = useWebSocket();

  const [board, setBoard] = useState<Board | null>(null);
  const [columns, setColumns] = useState<Column[]>([]);
  const [tasks, setTasks] = useState<Task[]>([]);
  const [loading, setLoading] = useState(true);
  const [isSettingsModalOpen, setIsSettingsModalOpen] = useState(false);

  // Tab State
  const [activeTab, setActiveTab] = useState<"kanban" | "timeline" | "sprint" | "calendar" | "analytics" | "ai">("kanban");

  // Selected Task for modal
  const [selectedTaskId, setSelectedTaskId] = useState<string | null>(null);

  const fetchData = async () => {
    if (!workspaceId || !boardId) return;
    try {
      const [boardData, colsData, tasksData] = await Promise.all([
        boardApi.get(workspaceId, boardId),
        boardApi.getColumns(workspaceId, boardId),
        taskApi.list(workspaceId, boardId),
      ]);
      setBoard(boardData);
      setColumns(colsData);
      setTasks(tasksData);
    } catch (err) {
      console.error(err);
      toast.error("Failed to load board details");
      router.push("/dashboard");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (authLoading) return;
    if (!isAuthenticated) return;
    fetchData();
  }, [workspaceId, boardId, authLoading, isAuthenticated]);

  // Set up WebSockets for realtime updates
  useEffect(() => {
    if (!boardId || !isConnected) return;

    // Join board websocket room
    joinRoom(boardId);

    // Listen to task status changes, moves, columns updates
    const unsubscribeTaskMoved = registerListener("task_moved", () => {
      fetchData();
      toast.success("Board updated in real-time");
    });
    const unsubscribeTaskCreated = registerListener("task_created", () => {
      fetchData();
    });
    const unsubscribeTaskUpdated = registerListener("task_updated", () => {
      fetchData();
    });
    const unsubscribeTaskDeleted = registerListener("task_deleted", () => {
      fetchData();
    });

    return () => {
      leaveRoom(boardId);
      unsubscribeTaskMoved();
      unsubscribeTaskCreated();
      unsubscribeTaskUpdated();
      unsubscribeTaskDeleted();
    };
  }, [boardId, isConnected]);

  const handleExportExcel = () => {
    // Direct link trigger to trigger excel download
    const token = localStorage.getItem("accessToken");
    const baseURL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080/api";
    window.location.href = `${baseURL}/workspaces/${workspaceId}/boards/${boardId}/export/excel?token=${token}`;
    toast.success("Exporting board tasks to Excel...");
  };

  const handleDuplicateBoard = async () => {
    try {
      const dup = await boardApi.duplicate(workspaceId, boardId);
      toast.success("Board duplicated successfully!");
      router.push(`/boards/${dup.id}?ws=${workspaceId}`);
    } catch (err) {
      toast.error("Failed to duplicate board");
    }
  };

  const handleArchiveBoard = async () => {
    if (!confirm("Are you sure you want to archive this board?")) return;
    try {
      await boardApi.archive(workspaceId, boardId);
      toast.success("Board archived successfully");
      router.push(`/workspaces/${workspaceId}`);
    } catch (err) {
      toast.error("Failed to archive board");
    }
  };

  if (authLoading || loading || !board) {
    return (
      <div className="min-h-screen bg-slate-950 text-slate-100 flex flex-col items-center justify-center font-sans">
        <div className="w-10 h-10 border-4 border-indigo-500/20 border-t-indigo-500 rounded-full animate-spin mb-4" />
        <p className="text-sm text-slate-400 font-light">Loading board details...</p>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-slate-950 text-slate-100 font-sans flex flex-col h-screen overflow-hidden">
      {/* Board Header Bar */}
      <header className="h-20 shrink-0 border-b border-slate-900 bg-slate-950/80 backdrop-blur-md px-8 md:px-12 flex items-center justify-between sticky top-0 z-40">
        <div className="flex items-center gap-4">
          <Link href={`/workspaces/${workspaceId}`} className="text-sm font-medium text-slate-400 hover:text-white transition">
            ← Workspace
          </Link>
          <span className="text-slate-700">/</span>
          <div className="flex items-center gap-2.5">
            <div className="w-3.5 h-3.5 rounded-full" style={{ backgroundColor: board.color }} />
            <h1 className="font-semibold text-white text-base">{board.name}</h1>
          </div>
          <span className="text-slate-800">|</span>
          <span className="text-[10px] text-slate-500 font-light hidden sm:inline">
            Status: {isConnected ? "Connected Real-Time" : "Offline / Reconnecting"}
          </span>
        </div>

        <div className="flex items-center gap-3">
          <button
            onClick={handleExportExcel}
            className="p-2 border border-slate-850 hover:bg-slate-900 hover:border-slate-800 rounded-xl text-slate-400 hover:text-white transition"
            title="Export Excel"
          >
            <FileSpreadsheet className="w-4 h-4" />
          </button>
          <button
            onClick={handleDuplicateBoard}
            className="p-2 border border-slate-850 hover:bg-slate-900 hover:border-slate-800 rounded-xl text-slate-400 hover:text-white transition"
            title="Duplicate Board"
          >
            <Copy className="w-4 h-4" />
          </button>
          <button
            onClick={handleArchiveBoard}
            className="p-2 border border-slate-850 hover:bg-slate-900 hover:border-slate-800 rounded-xl text-slate-400 hover:text-rose-400 transition"
            title="Archive Board"
          >
            <Archive className="w-4 h-4" />
          </button>
          <button
            onClick={() => setIsSettingsModalOpen(true)}
            className="p-2 border border-slate-850 hover:bg-slate-900 hover:border-slate-800 rounded-xl text-slate-400 hover:text-white transition"
            title="Board Settings"
          >
            <Settings className="w-4 h-4" />
          </button>
        </div>
      </header>

      {/* Board view tab selectors */}
      <div className="shrink-0 h-14 border-b border-slate-900 px-8 md:px-12 flex items-center justify-between bg-slate-950/40">
        <div className="flex gap-4">
          {[
            { id: "kanban", label: "Kanban Board", icon: <FolderKanban className="w-4 h-4" /> },
            { id: "timeline", label: "Timeline (Gantt)", icon: <Calendar className="w-4 h-4" /> },
            { id: "sprint", label: "Sprints Backlog", icon: <Layers className="w-4 h-4" /> },
            { id: "calendar", label: "Task Calendar", icon: <Calendar className="w-4 h-4" /> },
            { id: "analytics", label: "Analytics Dashboard", icon: <BarChart2 className="w-4 h-4" /> },
            { id: "ai", label: "AIAssistant", icon: <Sparkles className="w-4 h-4 text-indigo-400" /> },
          ].map((tab) => (
            <button
              key={tab.id}
              onClick={() => setActiveTab(tab.id as any)}
              className={`flex items-center gap-2 px-3 py-1.5 rounded-lg text-xs font-semibold border transition ${
                activeTab === tab.id
                  ? "bg-slate-900 border-slate-850 text-white"
                  : "bg-transparent border-transparent text-slate-500 hover:text-slate-350"
              }`}
            >
              {tab.icon}
              {tab.label}
            </button>
          ))}
        </div>
      </div>

      {/* Main Board Workspace Panel (scrollable) */}
      <main className="flex-1 overflow-auto p-8 md:p-12 min-h-0 bg-slate-950/10">
        {activeTab === "kanban" && (
          <KanbanBoard
            workspaceId={workspaceId}
            boardId={boardId}
            tasks={tasks}
            columns={columns}
            onRefresh={fetchData}
            onSelectTask={setSelectedTaskId}
          />
        )}

        {activeTab === "timeline" && (
          <TimelineView tasks={tasks} onSelectTask={setSelectedTaskId} />
        )}

        {activeTab === "sprint" && (
          <SprintView
            workspaceId={workspaceId}
            boardId={boardId}
            tasks={tasks}
            onRefresh={fetchData}
          />
        )}

        {activeTab === "calendar" && (
          <CalendarView tasks={tasks} onSelectTask={setSelectedTaskId} />
        )}

        {activeTab === "analytics" && (
          <AnalyticsDashboard tasks={tasks} columns={columns} />
        )}

        {activeTab === "ai" && (
          <AIAssistant
            workspaceId={workspaceId}
            boardId={boardId}
            onRefresh={fetchData}
          />
        )}
      </main>

      {/* RENDER SELECTED TASK MODAL */}
      {selectedTaskId && (
        <TaskModal
          workspaceId={workspaceId}
          boardId={boardId}
          taskId={selectedTaskId}
          onClose={() => setSelectedTaskId(null)}
          onUpdate={fetchData}
        />
      )}

      {/* RENDER BOARD SETTINGS MODAL */}
      {isSettingsModalOpen && (
        <BoardSettingsModal
          workspaceId={workspaceId}
          board={board}
          columns={columns}
          onClose={() => setIsSettingsModalOpen(false)}
          onRefresh={fetchData}
        />
      )}
    </div>
  );
}
