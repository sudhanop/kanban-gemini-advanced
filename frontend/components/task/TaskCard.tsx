"use client";

import { Task } from "@/lib/api";
import { Clock, CheckSquare, MessageSquare, AlertCircle } from "lucide-react";

interface TaskCardProps {
  task: Task;
  onClick: () => void;
  onDragStart: (e: React.DragEvent, taskId: string) => void;
  onDragEnd?: (e: React.DragEvent) => void;
}

export function TaskCard({ task, onClick, onDragStart, onDragEnd }: TaskCardProps) {
  const getPriorityColor = (prio: string) => {
    switch (prio.toLowerCase()) {
      case "high":
        return "bg-rose-500/10 border-rose-500/20 text-rose-400";
      case "medium":
        return "bg-amber-500/10 border-amber-500/20 text-amber-400";
      default:
        return "bg-emerald-500/10 border-emerald-500/20 text-emerald-400";
    }
  };

  const completedSubtasks = task.subtasks?.filter((s) => s.is_completed).length || 0;
  const totalSubtasks = task.subtasks?.length || 0;

  return (
    <div
      draggable
      onDragStart={(e) => onDragStart(e, task.id)}
      onDragEnd={onDragEnd}
      onClick={onClick}
      className="group p-4 rounded-xl border border-slate-900 bg-slate-950/60 backdrop-blur-sm hover:border-slate-800 hover:bg-slate-900/60 shadow-md transition duration-200 cursor-grab active:cursor-grabbing text-left space-y-3"
    >
      <div className="flex justify-between items-start gap-2">
        <span className={`text-[10px] uppercase font-semibold px-2 py-0.5 border rounded-md ${getPriorityColor(task.priority || "low")}`}>
          {task.priority || "low"}
        </span>
        {task.story_points > 0 && (
          <span className="text-[10px] font-semibold text-slate-500 bg-slate-900 px-1.5 py-0.5 rounded-md border border-slate-850">
            {task.story_points} pts
          </span>
        )}
      </div>

      <h4 className="font-semibold text-slate-200 group-hover:text-white transition text-sm leading-snug line-clamp-2">
        {task.title}
      </h4>

      {task.description && (
        <p className="text-xs text-slate-500 font-light line-clamp-2 leading-relaxed">
          {task.description}
        </p>
      )}

      {/* Task Meta Footer */}
      <div className="flex items-center justify-between text-[11px] text-slate-500 pt-2 border-t border-slate-900/50">
        <div className="flex items-center gap-3">
          {task.due_date && (
            <span className="flex items-center gap-1 text-slate-400">
              <Clock className="w-3.5 h-3.5" />
              {new Date(task.due_date).toLocaleDateString([], { month: "short", day: "numeric" })}
            </span>
          )}
          {totalSubtasks > 0 && (
            <span className="flex items-center gap-1">
              <CheckSquare className="w-3.5 h-3.5" />
              {completedSubtasks}/{totalSubtasks}
            </span>
          )}
          {task.comments && task.comments.length > 0 && (
            <span className="flex items-center gap-1">
              <MessageSquare className="w-3.5 h-3.5" />
              {task.comments.length}
            </span>
          )}
        </div>

        {/* Assignee Avatars */}
        <div className="flex -space-x-1.5 overflow-hidden">
          {task.assignees?.map((a) => (
            <img
              key={a.id}
              src={a.user?.avatar || "https://images.unsplash.com/photo-1535713875002-d1d0cf377fde"}
              alt="avatar"
              title={a.user?.name}
              className="w-5.5 h-5.5 rounded-full object-cover ring-1 ring-slate-900"
            />
          ))}
        </div>
      </div>
    </div>
  );
}
