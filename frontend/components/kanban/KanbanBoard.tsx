"use client";

import { useEffect, useState } from "react";
import { Column, Task, boardApi, taskApi } from "@/lib/api";
import { TaskCard } from "@/components/task/TaskCard";
import { Plus, Trash2, Edit2, AlertCircle } from "lucide-react";
import toast from "react-hot-toast";

interface KanbanBoardProps {
  workspaceId: string;
  boardId: string;
  tasks: Task[];
  columns: Column[];
  onRefresh: () => void;
  onSelectTask: (taskId: string) => void;
}

export function KanbanBoard({
  workspaceId,
  boardId,
  tasks,
  columns,
  onRefresh,
  onSelectTask,
}: KanbanBoardProps) {
  const [newColName, setNewColName] = useState("");
  const [showColInput, setShowColInput] = useState(false);

  // Task creation states in columns
  const [colTaskInput, setColTaskInput] = useState<Record<string, string>>({});

  const handleDragStart = (e: React.DragEvent, taskId: string) => {
    e.dataTransfer.setData("text/plain", taskId);
  };

  const handleDragOver = (e: React.DragEvent) => {
    e.preventDefault();
  };

  const handleDrop = async (e: React.DragEvent, targetColumnId: string) => {
    e.preventDefault();
    const taskId = e.dataTransfer.getData("text/plain");
    if (!taskId) return;

    try {
      // Find the task inside current state
      const task = tasks.find((t) => t.id === taskId);
      if (!task || task.column_id === targetColumnId) return;

      // Optimistically move task
      await taskApi.move(workspaceId, boardId, taskId, {
        column_id: targetColumnId,
        position: 0, // Put at the top
      });
      toast.success("Task moved");
      onRefresh();
    } catch (err) {
      toast.error("Failed to move task");
    }
  };

  const handleCreateColumn = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newColName.trim()) return;
    try {
      await boardApi.createColumn(workspaceId, boardId, { name: newColName });
      setNewColName("");
      setShowColInput(false);
      toast.success("Column added");
      onRefresh();
    } catch (err) {
      toast.error("Failed to add column");
    }
  };

  const handleDeleteColumn = async (columnId: string) => {
    if (!confirm("Are you sure you want to delete this column?")) return;
    try {
      await boardApi.deleteColumn(workspaceId, boardId, columnId);
      toast.success("Column deleted");
      onRefresh();
    } catch (err) {
      toast.error("Failed to delete column");
    }
  };

  const handleAddTask = async (columnId: string) => {
    const title = colTaskInput[columnId];
    if (!title?.trim()) return;
    try {
      await taskApi.create(workspaceId, boardId, {
        title,
        column_id: columnId,
        priority: "medium",
        status: "todo",
      });
      setColTaskInput((prev) => ({ ...prev, [columnId]: "" }));
      toast.success("Task created");
      onRefresh();
    } catch (err) {
      toast.error("Failed to create task");
    }
  };

  return (
    <div className="flex gap-6 overflow-x-auto pb-6 items-start h-full min-h-[60vh] max-w-full">
      {columns.map((col) => {
        const colTasks = tasks.filter((t) => t.column_id === col.id) || [];
        const isWipExceeded = col.wip_limit > 0 && colTasks.length > col.wip_limit;

        return (
          <div
            key={col.id}
            onDragOver={handleDragOver}
            onDrop={(e) => handleDrop(e, col.id)}
            className="w-80 shrink-0 flex flex-col max-h-[80vh] border border-slate-900 bg-slate-950/40 backdrop-blur-md rounded-2xl p-4 space-y-4"
          >
            {/* Column Header */}
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2 overflow-hidden">
                <h3 className="font-semibold text-slate-200 truncate text-sm">{col.name}</h3>
                <span className={`text-[10px] px-2 py-0.5 rounded-full font-bold border ${
                  isWipExceeded
                    ? "bg-rose-500/15 border-rose-500/20 text-rose-400"
                    : "bg-slate-900 border-slate-850 text-slate-400"
                }`}>
                  {colTasks.length} {col.wip_limit > 0 && `/ ${col.wip_limit}`}
                </span>
              </div>
              <button
                onClick={() => handleDeleteColumn(col.id)}
                className="p-1 text-slate-600 hover:text-rose-400 transition"
              >
                <Trash2 className="w-3.5 h-3.5" />
              </button>
            </div>

            {/* WIP warning banner */}
            {isWipExceeded && (
              <div className="flex items-center gap-1.5 p-2.5 rounded-xl border border-rose-500/10 bg-rose-500/5 text-[10px] text-rose-400">
                <AlertCircle className="w-3.5 h-3.5 shrink-0" />
                <span>WIP limit reached! Limit is {col.wip_limit}.</span>
              </div>
            )}

            {/* Add Task input in column */}
            <div className="flex gap-1.5 shrink-0">
              <input
                type="text"
                value={colTaskInput[col.id] || ""}
                onChange={(e) =>
                  setColTaskInput((prev) => ({ ...prev, [col.id]: e.target.value }))
                }
                placeholder="New task..."
                className="flex-1 px-3 py-1.5 rounded-xl border border-slate-900 bg-slate-900/20 text-xs text-white focus:outline-none focus:border-indigo-500 transition placeholder-slate-700"
                onKeyDown={(e) => e.key === "Enter" && handleAddTask(col.id)}
              />
              <button
                onClick={() => handleAddTask(col.id)}
                className="p-1.5 bg-indigo-650/10 border border-indigo-500/10 text-indigo-400 hover:bg-indigo-600 hover:text-white rounded-xl transition text-xs shrink-0"
              >
                Add
              </button>
            </div>

            {/* Tasks list */}
            <div className="flex-1 overflow-y-auto space-y-3 pr-1 min-h-[150px]">
              {colTasks.map((task) => (
                <TaskCard
                  key={task.id}
                  task={task}
                  onClick={() => onSelectTask(task.id)}
                  onDragStart={(e) => handleDragStart(e, task.id)}
                />
              ))}
            </div>
          </div>
        );
      })}

      {/* Create Column */}
      <div className="w-80 shrink-0">
        {showColInput ? (
          <form
            onSubmit={handleCreateColumn}
            className="p-4 border border-slate-850 bg-slate-950/20 rounded-2xl space-y-3"
          >
            <input
              type="text"
              value={newColName}
              onChange={(e) => setNewColName(e.target.value)}
              placeholder="Column title..."
              className="w-full px-3 py-2 rounded-xl border border-slate-900 bg-slate-900/40 text-xs text-white focus:outline-none focus:border-indigo-500 transition"
              autoFocus
            />
            <div className="flex justify-end gap-2.5">
              <button
                type="button"
                onClick={() => setShowColInput(false)}
                className="px-3 py-1.5 text-xs text-slate-400 hover:text-white transition"
              >
                Cancel
              </button>
              <button
                type="submit"
                className="px-3 py-1.5 text-xs font-semibold bg-indigo-600 hover:bg-indigo-500 text-white rounded-xl shadow-md transition"
              >
                Add Column
              </button>
            </div>
          </form>
        ) : (
          <button
            onClick={() => setShowColInput(true)}
            className="w-full flex items-center justify-center gap-2 p-4 rounded-2xl border border-dashed border-slate-900 hover:border-slate-850 hover:bg-slate-950/20 text-slate-500 hover:text-slate-300 transition duration-200"
          >
            <Plus className="w-4 h-4" /> Add Column
          </button>
        )}
      </div>
    </div>
  );
}
