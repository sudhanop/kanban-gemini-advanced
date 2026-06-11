"use client";

import { Task } from "@/lib/api";
import { Calendar, AlertCircle } from "lucide-react";

interface TimelineViewProps {
  tasks: Task[];
  onSelectTask: (taskId: string) => void;
}

export function TimelineView({ tasks, onSelectTask }: TimelineViewProps) {
  // Sort tasks by due date
  const sortedTasks = [...tasks].sort((a, b) => {
    const da = a.due_date ? new Date(a.due_date).getTime() : 0;
    const db = b.due_date ? new Date(b.due_date).getTime() : 0;
    return da - db;
  });

  const getPriorityBorder = (prio: string) => {
    switch (prio.toLowerCase()) {
      case "high":
        return "border-l-rose-500 hover:border-rose-500 bg-rose-500/5";
      case "medium":
        return "border-l-amber-500 hover:border-amber-500 bg-amber-500/5";
      default:
        return "border-l-emerald-500 hover:border-emerald-500 bg-emerald-500/5";
    }
  };

  return (
    <div className="border border-slate-900 bg-slate-950/45 backdrop-blur-md rounded-2xl p-6 space-y-6">
      <div className="flex items-center justify-between pb-4 border-b border-slate-900/60">
        <div>
          <h3 className="font-semibold text-slate-200 text-sm">Gantt Timeline</h3>
          <p className="text-[11px] text-slate-500 font-light mt-0.5">Visually track task durations, milestones, and scheduling schedules.</p>
        </div>
      </div>

      {sortedTasks.length === 0 ? (
        <div className="p-8 text-center text-xs text-slate-500">No scheduled tasks. Add due dates to tasks to view them on the timeline.</div>
      ) : (
        <div className="space-y-3">
          {sortedTasks.map((t) => {
            const start = t.start_date ? new Date(t.start_date) : new Date(t.created_at);
            const due = t.due_date ? new Date(t.due_date) : null;
            
            return (
              <div
                key={t.id}
                onClick={() => onSelectTask(t.id)}
                className={`p-4 rounded-xl border border-slate-900 border-l-4 hover:border-slate-800 transition cursor-pointer flex flex-col md:flex-row justify-between md:items-center gap-4 ${getPriorityBorder(t.priority || "low")}`}
              >
                <div>
                  <h4 className="font-semibold text-slate-200 text-sm">{t.title}</h4>
                  <p className="text-xs text-slate-500 font-light mt-0.5 line-clamp-1">{t.description || "No description provided."}</p>
                </div>

                <div className="flex items-center gap-3 text-xs shrink-0">
                  <div className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-slate-900 border border-slate-850 text-slate-400">
                    <Calendar className="w-3.5 h-3.5" />
                    <span>
                      {start.toLocaleDateString([], { month: "short", day: "numeric" })}
                      {due ? ` – ${due.toLocaleDateString([], { month: "short", day: "numeric" })}` : " (No due date)"}
                    </span>
                  </div>
                  <span className="text-[10px] uppercase font-semibold px-2 py-1 bg-slate-900 border border-slate-850 rounded-md text-slate-500">
                    {t.status || "todo"}
                  </span>
                </div>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
