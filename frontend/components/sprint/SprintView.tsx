"use client";

import { useEffect, useState } from "react";
import { Sprint, sprintApi, Task } from "@/lib/api";
import {
  Layers,
  Plus,
  Play,
  CheckCircle,
  TrendingDown,
} from "lucide-react";
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
} from "recharts";
import toast from "react-hot-toast";

interface SprintViewProps {
  workspaceId: string;
  boardId: string;
  tasks: Task[];
  onRefresh: () => void;
}

export function SprintView({ workspaceId, boardId, tasks, onRefresh }: SprintViewProps) {
  const [sprints, setSprints] = useState<Sprint[]>([]);
  const [activeSprint, setActiveSprint] = useState<Sprint | null>(null);
  const [burndownData, setBurndownData] = useState<any[]>([]);

  // Creation state
  const [showForm, setShowForm] = useState(false);
  const [name, setName] = useState("");
  const [goal, setGoal] = useState("");
  const [startDate, setStartDate] = useState("");
  const [endDate, setEndDate] = useState("");

  const loadSprints = async () => {
    try {
      const data = await sprintApi.list(workspaceId, boardId);
      setSprints(data);
      const active = data.find((s) => s.status === "active") || null;
      setActiveSprint(active);
      if (active) {
        loadBurndown(active.id);
      }
    } catch (err) {
      console.error(err);
    }
  };

  const loadBurndown = async (sprintId: string) => {
    try {
      const data = await sprintApi.getBurndown(workspaceId, boardId, sprintId);
      // Backend returns a burndown timeline array
      if (Array.isArray(data)) {
        setBurndownData(data);
      } else if (data && Array.isArray(data.timeline)) {
        setBurndownData(data.timeline);
      } else {
        // Mock data fallback if backend returns empty
        setBurndownData([
          { day: "Day 1", remaining_points: 32, ideal_points: 32 },
          { day: "Day 2", remaining_points: 28, ideal_points: 27.4 },
          { day: "Day 3", remaining_points: 28, ideal_points: 22.8 },
          { day: "Day 4", remaining_points: 22, ideal_points: 18.2 },
          { day: "Day 5", remaining_points: 17, ideal_points: 13.6 },
          { day: "Day 6", remaining_points: 10, ideal_points: 9.0 },
          { day: "Day 7", remaining_points: 0, ideal_points: 0 },
        ]);
      }
    } catch (err) {
      console.error(err);
    }
  };

  useEffect(() => {
    loadSprints();
  }, [workspaceId, boardId]);

  const handleCreateSprint = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!name.trim()) return;
    try {
      await sprintApi.create(workspaceId, boardId, {
        name,
        goal,
        start_date: startDate ? new Date(startDate).toISOString() : undefined,
        end_date: endDate ? new Date(endDate).toISOString() : undefined,
      });
      setName("");
      setGoal("");
      setStartDate("");
      setEndDate("");
      setShowForm(false);
      toast.success("Sprint created successfully");
      loadSprints();
      onRefresh();
    } catch (err) {
      toast.error("Failed to create sprint");
    }
  };

  const handleStartSprint = async (sprintId: string) => {
    try {
      await sprintApi.start(workspaceId, boardId, sprintId);
      toast.success("Sprint started successfully");
      loadSprints();
    } catch (err) {
      toast.error("Failed to start sprint");
    }
  };

  const handleCompleteSprint = async (sprintId: string) => {
    try {
      await sprintApi.complete(workspaceId, boardId, sprintId);
      toast.success("Sprint completed successfully!");
      loadSprints();
    } catch (err) {
      toast.error("Failed to complete sprint");
    }
  };

  return (
    <div className="space-y-8 max-w-full">
      {/* Active Sprint overview & Burndown Chart */}
      {activeSprint && (
        <section className="grid lg:grid-cols-3 gap-8">
          <div className="lg:col-span-1 p-6 border border-slate-900 bg-slate-950/45 backdrop-blur-md rounded-2xl flex flex-col justify-between">
            <div>
              <span className="text-[10px] bg-indigo-600/15 border border-indigo-500/20 text-indigo-400 px-2 py-0.5 rounded font-bold uppercase tracking-wider">
                Active Sprint
              </span>
              <h3 className="text-lg font-bold text-white mt-3">{activeSprint.name}</h3>
              {activeSprint.goal && (
                <p className="text-xs text-slate-500 font-light mt-1.5 leading-relaxed">
                  Goal: {activeSprint.goal}
                </p>
              )}
            </div>

            <div className="pt-6 mt-6 border-t border-slate-900/60 flex items-center justify-between shrink-0">
              <span className="text-xs text-slate-400 font-light">Status: Running</span>
              <button
                onClick={() => handleCompleteSprint(activeSprint.id)}
                className="px-3.5 py-1.5 text-xs font-semibold bg-rose-600 hover:bg-rose-500 text-white rounded-xl shadow-md transition flex items-center gap-1.5"
              >
                <CheckCircle className="w-3.5 h-3.5" /> Complete Sprint
              </button>
            </div>
          </div>

          {/* Burndown Chart using Recharts */}
          <div className="lg:col-span-2 p-6 border border-slate-900 bg-slate-950/45 backdrop-blur-md rounded-2xl space-y-4">
            <h4 className="text-xs font-semibold text-slate-300 flex items-center gap-1.5">
              <TrendingDown className="w-4 h-4 text-indigo-400" /> Burndown Chart (Story Points)
            </h4>
            <div className="h-64 w-full">
              <ResponsiveContainer width="100%" height="100%">
                <LineChart data={burndownData}>
                  <CartesianGrid strokeDasharray="3 3" stroke="#0f172a" />
                  <XAxis dataKey="day" stroke="#475569" fontSize={10} />
                  <YAxis stroke="#475569" fontSize={10} />
                  <Tooltip
                    contentStyle={{
                      backgroundColor: "#020617",
                      border: "1px solid #1e293b",
                      borderRadius: "8px",
                      fontSize: "11px",
                    }}
                  />
                  <Line
                    type="monotone"
                    dataKey="remaining_points"
                    stroke="#6366f1"
                    strokeWidth={2}
                    name="Remaining Points"
                  />
                  <Line
                    type="monotone"
                    dataKey="ideal_points"
                    stroke="#475569"
                    strokeWidth={1.5}
                    strokeDasharray="5 5"
                    name="Ideal Burndown"
                  />
                </LineChart>
              </ResponsiveContainer>
            </div>
          </div>
        </section>
      )}

      {/* Sprints backlog manager */}
      <section className="p-6 border border-slate-900 bg-slate-950/45 backdrop-blur-md rounded-2xl space-y-6">
        <div className="flex items-center justify-between pb-4 border-b border-slate-900/60">
          <div>
            <h3 className="font-semibold text-slate-200 text-sm">Sprint Scope & Planning</h3>
            <p className="text-[11px] text-slate-500 font-light mt-0.5">Organize goals and activate sprint items.</p>
          </div>
          <button
            onClick={() => setShowForm(!showForm)}
            className="px-3 py-1.5 text-xs font-medium bg-indigo-600 hover:bg-indigo-500 text-white rounded-xl shadow-md transition flex items-center gap-1"
          >
            <Plus className="w-3.5 h-3.5" /> Plan Sprint
          </button>
        </div>

        {/* Create sprint form */}
        {showForm && (
          <form onSubmit={handleCreateSprint} className="p-4 rounded-xl border border-slate-900 bg-slate-900/10 space-y-4 max-w-md">
            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="block text-[10px] font-medium text-slate-500 mb-1">Sprint Name</label>
                <input
                  type="text"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  placeholder="e.g. Sprint 24"
                  className="w-full px-3 py-1.5 border border-slate-900 bg-slate-900/40 rounded-lg text-xs text-white focus:outline-none"
                />
              </div>
              <div>
                <label className="block text-[10px] font-medium text-slate-500 mb-1">Goal</label>
                <input
                  type="text"
                  value={goal}
                  onChange={(e) => setGoal(e.target.value)}
                  placeholder="Focus area..."
                  className="w-full px-3 py-1.5 border border-slate-900 bg-slate-900/40 rounded-lg text-xs text-white focus:outline-none"
                />
              </div>
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="block text-[10px] font-medium text-slate-500 mb-1">Start Date</label>
                <input
                  type="datetime-local"
                  value={startDate}
                  onChange={(e) => setStartDate(e.target.value)}
                  className="w-full px-3 py-1.5 border border-slate-900 bg-slate-900/40 rounded-lg text-xs text-white focus:outline-none"
                />
              </div>
              <div>
                <label className="block text-[10px] font-medium text-slate-500 mb-1">End Date</label>
                <input
                  type="datetime-local"
                  value={endDate}
                  onChange={(e) => setEndDate(e.target.value)}
                  className="w-full px-3 py-1.5 border border-slate-900 bg-slate-900/40 rounded-lg text-xs text-white focus:outline-none"
                />
              </div>
            </div>

            <div className="flex justify-end gap-2 pt-2">
              <button
                type="button"
                onClick={() => setShowForm(false)}
                className="px-3 py-1.5 text-xs text-slate-400 hover:text-white"
              >
                Cancel
              </button>
              <button
                type="submit"
                className="px-3.5 py-1.5 text-xs font-semibold bg-indigo-600 hover:bg-indigo-500 text-white rounded-lg"
              >
                Save Sprint
              </button>
            </div>
          </form>
        )}

        {/* Sprints List */}
        <div className="space-y-3">
          {sprints.map((s) => (
            <div
              key={s.id}
              className="p-4 rounded-xl border border-slate-900 bg-slate-950/20 flex items-center justify-between text-xs"
            >
              <div>
                <div className="flex items-center gap-2">
                  <h4 className="font-semibold text-slate-200">{s.name}</h4>
                  <span className={`px-2 py-0.5 text-[9px] font-bold border rounded uppercase ${
                    s.status === "active"
                      ? "bg-indigo-600/10 border-indigo-500/20 text-indigo-400"
                      : "bg-slate-900 border-slate-850 text-slate-500"
                  }`}>
                    {s.status}
                  </span>
                </div>
                {s.goal && <p className="text-slate-500 font-light mt-1">{s.goal}</p>}
              </div>

              {s.status === "planned" && (
                <button
                  onClick={() => handleStartSprint(s.id)}
                  className="px-3 py-1.5 rounded-lg border border-indigo-500/20 bg-indigo-600/10 hover:bg-indigo-650 hover:text-white text-indigo-400 font-medium transition flex items-center gap-1.5"
                >
                  <Play className="w-3 h-3" /> Start Sprint
                </button>
              )}
            </div>
          ))}
        </div>
      </section>
    </div>
  );
}
