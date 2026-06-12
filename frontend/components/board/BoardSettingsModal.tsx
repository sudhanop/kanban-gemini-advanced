"use client";

import { useState, useEffect } from "react";
import { boardApi, Board, Column } from "@/lib/api";
import { Settings, X, Plus, Trash2, ArrowUp, ArrowDown, Columns, AlertCircle } from "lucide-react";
import toast from "react-hot-toast";
import { useRouter } from "next/navigation";

interface BoardSettingsModalProps {
  workspaceId: string;
  board: Board;
  columns: Column[];
  onClose: () => void;
  onRefresh: () => void;
}

export function BoardSettingsModal({
  workspaceId,
  board,
  columns,
  onClose,
  onRefresh,
}: BoardSettingsModalProps) {
  const router = useRouter();
  const [boardName, setBoardName] = useState(board.name);
  const [boardDesc, setBoardDesc] = useState(board.description || "");
  const [boardColor, setBoardColor] = useState(board.color);
  
  // Column list state
  const [localColumns, setLocalColumns] = useState<Column[]>([]);
  const [newColName, setNewColName] = useState("");
  
  // Track inputs for renaming and WIP limits
  const [colNames, setColNames] = useState<Record<string, string>>({});
  const [colWips, setColWips] = useState<Record<string, number>>({});

  useEffect(() => {
    // Sort columns by order index locally
    const sorted = [...columns].sort((a, b) => a.position - b.position || 0);
    setLocalColumns(sorted);
    
    // Set initial form states
    const names: Record<string, string> = {};
    const wips: Record<string, number> = {};
    sorted.forEach((c) => {
      names[c.id] = c.name;
      wips[c.id] = c.wip_limit || 0;
    });
    setColNames(names);
    setColWips(wips);
  }, [columns]);

  const handleUpdateBoard = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!boardName.trim()) return;
    try {
      await boardApi.update(workspaceId, board.id, {
        name: boardName,
        description: boardDesc,
        color: boardColor,
      });
      toast.success("Board updated successfully!");
      onRefresh();
      onClose();
    } catch (err) {
      toast.error("Failed to update board");
    }
  };

  const handleDeleteBoard = async () => {
    if (!confirm("Are you sure you want to permanently delete this board? This action cannot be undone.")) return;
    try {
      await boardApi.delete(workspaceId, board.id);
      toast.success("Board deleted");
      router.push(`/workspaces/${workspaceId}`);
    } catch (err) {
      toast.error("Failed to delete board");
    }
  };

  const handleAddColumn = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newColName.trim()) return;
    try {
      await boardApi.createColumn(workspaceId, board.id, { name: newColName });
      setNewColName("");
      toast.success("Column added");
      onRefresh();
    } catch (err) {
      toast.error("Failed to add column");
    }
  };

  const handleDeleteColumn = async (columnId: string) => {
    if (localColumns.length <= 1) {
      toast.error("A board must have at least one column");
      return;
    }
    if (!confirm("Delete this column? Existing tasks will be moved to the first column.")) return;
    try {
      await boardApi.deleteColumn(workspaceId, board.id, columnId);
      toast.success("Column deleted");
      onRefresh();
    } catch (err) {
      toast.error("Failed to delete column");
    }
  };

  const handleUpdateColumn = async (columnId: string) => {
    const name = colNames[columnId];
    const wipLimit = colWips[columnId];
    if (!name?.trim()) {
      toast.error("Column name is required");
      return;
    }
    try {
      await boardApi.updateColumn(workspaceId, board.id, columnId, {
        name,
        wip_limit: wipLimit,
      });
      toast.success("Column details saved");
      onRefresh();
    } catch (err) {
      toast.error("Failed to update column");
    }
  };

  const moveColumn = async (index: number, direction: "up" | "down") => {
    const newCols = [...localColumns];
    const targetIndex = direction === "up" ? index - 1 : index + 1;
    if (targetIndex < 0 || targetIndex >= newCols.length) return;

    // Swap items
    const temp = newCols[index];
    newCols[index] = newCols[targetIndex];
    newCols[targetIndex] = temp;

    setLocalColumns(newCols);

    try {
      const payload = newCols.map((c, i) => ({ id: c.id, order_index: i }));
      await boardApi.reorderColumns(workspaceId, board.id, payload);
      toast.success("Column order updated");
      onRefresh();
    } catch (err) {
      toast.error("Failed to reorder columns");
    }
  };

  return (
    <div className="fixed inset-0 bg-black/60 backdrop-blur-sm flex items-center justify-center z-50 p-4">
      <div className="w-full max-w-2xl p-6 rounded-2xl border border-slate-800 bg-slate-950 shadow-2xl flex flex-col max-h-[90vh] overflow-y-auto">
        <div className="flex items-center justify-between mb-6 pb-2 border-b border-slate-900">
          <h3 className="text-lg font-bold text-white flex items-center gap-2">
            <Settings className="w-5 h-5 text-indigo-400" /> Board Settings
          </h3>
          <button onClick={onClose} className="text-slate-400 hover:text-white transition">
            <X className="w-5 h-5" />
          </button>
        </div>

        <div className="grid md:grid-cols-2 gap-8">
          {/* Left Column: Board Metadata */}
          <div className="space-y-6">
            <h4 className="text-sm font-semibold text-slate-350">General Details</h4>
            <form onSubmit={handleUpdateBoard} className="space-y-4">
              <div>
                <label className="block text-xs text-slate-400 font-medium mb-1.5">Board Title</label>
                <input
                  type="text"
                  value={boardName}
                  onChange={(e) => setBoardName(e.target.value)}
                  className="w-full px-3.5 py-2.5 rounded-xl border border-slate-800 bg-slate-900 text-sm text-white focus:outline-none focus:border-indigo-500 transition"
                  placeholder="Board Name"
                  required
                />
              </div>

              <div>
                <label className="block text-xs text-slate-400 font-medium mb-1.5">Description (optional)</label>
                <textarea
                  value={boardDesc}
                  onChange={(e) => setBoardDesc(e.target.value)}
                  className="w-full h-24 px-3.5 py-2.5 rounded-xl border border-slate-800 bg-slate-900 text-sm text-white focus:outline-none focus:border-indigo-500 transition resize-none"
                  placeholder="Board goal..."
                />
              </div>

              <div>
                <label className="block text-xs text-slate-400 font-medium mb-1.5">Accent Color</label>
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

              <div className="flex gap-3 pt-4 border-t border-slate-900/60">
                <button
                  type="button"
                  onClick={handleDeleteBoard}
                  className="px-4 py-2 rounded-xl text-sm bg-rose-600/10 border border-rose-500/20 text-rose-400 hover:bg-rose-650 hover:text-white transition"
                >
                  Delete Board
                </button>
                <button
                  type="submit"
                  className="flex-1 px-4 py-2 rounded-xl text-sm bg-indigo-600 hover:bg-indigo-500 text-white font-medium transition"
                >
                  Save Board
                </button>
              </div>
            </form>
          </div>

          {/* Right Column: Column Management */}
          <div className="space-y-6 border-t md:border-t-0 md:border-l md:pl-8 border-slate-900/60">
            <h4 className="text-sm font-semibold text-slate-350 flex items-center gap-2">
              <Columns className="w-4 h-4 text-purple-400" /> Columns & WIP Limits
            </h4>

            {/* Add Column */}
            <form onSubmit={handleAddColumn} className="flex gap-2">
              <input
                type="text"
                value={newColName}
                onChange={(e) => setNewColName(e.target.value)}
                placeholder="New column name..."
                className="flex-1 px-3 py-1.5 rounded-xl border border-slate-800 bg-slate-900 text-xs text-white focus:outline-none focus:border-indigo-500 transition"
              />
              <button
                type="submit"
                className="px-3 py-1.5 text-xs bg-indigo-600 hover:bg-indigo-500 text-white font-medium rounded-xl transition flex items-center gap-1 shrink-0"
              >
                <Plus className="w-3.5 h-3.5" /> Add
              </button>
            </form>

            {/* Column List */}
            <div className="space-y-3 max-h-[350px] overflow-y-auto pr-1">
              {localColumns.map((col, index) => (
                <div key={col.id} className="p-3 rounded-xl border border-slate-900 bg-slate-900/10 text-xs space-y-2.5">
                  <div className="flex items-center gap-2 justify-between">
                    <input
                      type="text"
                      value={colNames[col.id] || ""}
                      onChange={(e) => setColNames({ ...colNames, [col.id]: e.target.value })}
                      className="bg-transparent text-slate-200 border-b border-transparent focus:border-indigo-500/50 hover:bg-slate-900/30 px-1 py-0.5 rounded transition font-semibold w-40"
                    />
                    <div className="flex items-center gap-1">
                      <button
                        type="button"
                        onClick={() => moveColumn(index, "up")}
                        disabled={index === 0}
                        className="p-1 hover:text-white text-slate-500 disabled:opacity-20 disabled:hover:text-slate-500 transition"
                        title="Move Up"
                      >
                        <ArrowUp className="w-3.5 h-3.5" />
                      </button>
                      <button
                        type="button"
                        onClick={() => moveColumn(index, "down")}
                        disabled={index === localColumns.length - 1}
                        className="p-1 hover:text-white text-slate-500 disabled:opacity-20 disabled:hover:text-slate-500 transition"
                        title="Move Down"
                      >
                        <ArrowDown className="w-3.5 h-3.5" />
                      </button>
                      <button
                        type="button"
                        onClick={() => handleDeleteColumn(col.id)}
                        className="p-1 text-slate-600 hover:text-rose-400 transition"
                        title="Delete Column"
                      >
                        <Trash2 className="w-3.5 h-3.5" />
                      </button>
                    </div>
                  </div>

                  <div className="flex items-center justify-between gap-4">
                    <div className="flex items-center gap-1.5">
                      <span className="text-[10px] text-slate-500">WIP Limit:</span>
                      <input
                        type="number"
                        min="0"
                        value={colWips[col.id] === 0 ? "" : colWips[col.id] || ""}
                        onChange={(e) => setColWips({ ...colWips, [col.id]: parseInt(e.target.value) || 0 })}
                        placeholder="None"
                        className="w-12 px-1.5 py-0.5 rounded border border-slate-800 bg-slate-900 text-[10px] text-center text-white focus:outline-none focus:border-indigo-500"
                      />
                    </div>
                    <button
                      type="button"
                      onClick={() => handleUpdateColumn(col.id)}
                      className="px-2 py-0.5 border border-indigo-500/20 bg-indigo-600/10 text-indigo-400 hover:bg-indigo-600 hover:text-white rounded transition text-[10px] font-medium"
                    >
                      Save
                    </button>
                  </div>
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
