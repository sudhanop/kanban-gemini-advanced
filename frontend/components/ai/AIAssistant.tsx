"use client";

import { useState } from "react";
import { aiApi } from "@/lib/api";
import { Sparkles, CheckSquare, Plus, AlertCircle, AlertTriangle, ShieldAlert } from "lucide-react";
import toast from "react-hot-toast";

interface AIAssistantProps {
  workspaceId: string;
  boardId: string;
  onRefresh: () => void;
}

export function AIAssistant({ workspaceId, boardId, onRefresh }: AIAssistantProps) {
  const [loading, setLoading] = useState(false);
  const [activeTab, setActiveTab] = useState<"suggest" | "analyze" | "sprint">("suggest");

  // Suggest tasks states
  const [prompt, setPrompt] = useState("");
  const [suggestions, setSuggestions] = useState<any[]>([]);
  const [suggestionId, setSuggestionId] = useState("");
  const [selectedIndices, setSelectedIndices] = useState<Record<number, boolean>>({});

  // Analysis states
  const [analysis, setAnalysis] = useState<any>(null);

  // Sprint recommendation states
  const [sprintRec, setSprintRec] = useState<any>(null);

  const handleSuggestTasks = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!prompt.trim()) return;
    setLoading(true);
    setSuggestions([]);
    try {
      const res = await aiApi.suggestTasks(workspaceId, boardId, prompt);
      
      // Handle response string format vs object format
      let list = [];
      if (typeof res.suggestions === "string") {
        try {
          list = JSON.parse(res.suggestions);
        } catch {
          list = [];
        }
      } else if (Array.isArray(res.suggestions)) {
        list = res.suggestions;
      }
      
      setSuggestions(list);
      setSuggestionId(res.suggestion_id || "suggested");
      
      // Preselect all
      const indices: Record<number, boolean> = {};
      list.forEach((_: any, i: number) => {
        indices[i] = true;
      });
      setSelectedIndices(indices);
      toast.success("Suggestions loaded. Please confirm to add.");
    } catch (err) {
      toast.error("Failed to suggest tasks");
    } finally {
      setLoading(false);
    }
  };

  const handleConfirmTasks = async () => {
    setLoading(true);
    try {
      const indices = Object.entries(selectedIndices)
        .filter(([_, checked]) => checked)
        .map(([idx]) => Number(idx));

      const tasksToConfirm = suggestions.filter((_, idx) => selectedIndices[idx]);

      await aiApi.confirmTasks(workspaceId, boardId, suggestionId, indices);
      
      toast.success("Tasks approved and added!");
      setSuggestions([]);
      onRefresh();
    } catch (err) {
      toast.error("Failed to add confirmed tasks");
    } finally {
      setLoading(false);
    }
  };

  const handleAnalyzeBoard = async () => {
    setLoading(true);
    setAnalysis(null);
    try {
      const res = await aiApi.analyzeBoard(workspaceId, boardId);
      setAnalysis(res);
      toast.success("Board health analysis completed!");
    } catch (err) {
      toast.error("Failed to analyze board");
    } finally {
      setLoading(false);
    }
  };

  const handleSprintRec = async () => {
    setLoading(true);
    setSprintRec(null);
    try {
      const res = await aiApi.sprintRecommendation(workspaceId, boardId);
      setSprintRec(res);
      toast.success("Sprint scope recommended!");
    } catch (err) {
      toast.error("Failed to load sprint recommendation");
    } finally {
      setLoading(false);
    }
  };

  const toggleSelectIndex = (idx: number) => {
    setSelectedIndices((prev) => ({ ...prev, [idx]: !prev[idx] }));
  };

  return (
    <div className="border border-slate-900 bg-slate-950/45 backdrop-blur-md rounded-2xl p-6 space-y-6 flex flex-col h-full">
      {/* Header Tabs */}
      <div className="flex border-b border-slate-900 pb-4 shrink-0 justify-between items-center">
        <div className="flex gap-2">
          {(["suggest", "analyze", "sprint"] as const).map((tab) => (
            <button
              key={tab}
              onClick={() => setActiveTab(tab)}
              className={`px-3 py-1.5 rounded-lg text-xs font-semibold border transition ${
                activeTab === tab
                  ? "bg-indigo-600/10 border-indigo-500/20 text-indigo-400"
                  : "bg-transparent border-transparent text-slate-500 hover:text-slate-300"
              }`}
            >
              {tab === "suggest" ? "Suggest Tasks" : tab === "analyze" ? "Analyze Board" : "Sprint Rec"}
            </button>
          ))}
        </div>
        <div className="px-2.5 py-1 bg-slate-900 border border-slate-850 rounded-lg text-[10px] text-slate-500 flex items-center gap-1">
          <Sparkles className="w-3 h-3 text-indigo-400 animate-pulse" /> Gemini AI
        </div>
      </div>

      {/* Tab content */}
      <div className="flex-1 overflow-y-auto min-h-[300px]">
        {activeTab === "suggest" && (
          <div className="space-y-6">
            <form onSubmit={handleSuggestTasks} className="space-y-3">
              <label className="block text-xs font-medium text-slate-400">Describe the task goals</label>
              <textarea
                value={prompt}
                onChange={(e) => setPrompt(e.target.value)}
                placeholder="e.g. Set up authentication using NextAuth, add layout tests, and secure database callbacks..."
                className="w-full h-24 p-3 rounded-xl border border-slate-900 bg-slate-900/20 text-xs text-white focus:outline-none focus:border-indigo-500 transition resize-none placeholder-slate-700 font-light"
              />
              <button
                type="submit"
                disabled={loading}
                className="w-full py-2 bg-indigo-600 hover:bg-indigo-500 text-white font-medium rounded-xl text-xs transition"
              >
                {loading ? "Thinking..." : "Generate AI Suggestions"}
              </button>
            </form>

            {suggestions.length > 0 && (
              <div className="space-y-4 pt-4 border-t border-slate-900/60">
                <div className="flex items-center justify-between">
                  <span className="text-xs font-semibold text-slate-300">Confirm & Add Tasks</span>
                  <span className="text-[10px] text-slate-500 font-light">Select items to create</span>
                </div>

                <div className="space-y-3">
                  {suggestions.map((s, idx) => (
                    <div
                      key={idx}
                      className="p-3 rounded-xl border border-slate-900 bg-slate-900/10 flex items-start gap-3 text-xs"
                    >
                      <input
                        type="checkbox"
                        checked={!!selectedIndices[idx]}
                        onChange={() => toggleSelectIndex(idx)}
                        className="w-4 h-4 rounded border-slate-800 bg-slate-900 text-indigo-600 focus:ring-0 mt-0.5 cursor-pointer"
                      />
                      <div className="flex-1">
                        <div className="flex justify-between items-start">
                          <p className="font-semibold text-slate-200">{s.title}</p>
                          <span className="text-[9px] uppercase font-bold text-slate-500 px-1 border border-slate-800 rounded bg-slate-900">
                            {s.priority}
                          </span>
                        </div>
                        <p className="text-slate-400 font-light mt-1 text-[11px] leading-relaxed">{s.description}</p>
                      </div>
                    </div>
                  ))}
                </div>

                <button
                  onClick={handleConfirmTasks}
                  disabled={loading}
                  className="w-full py-2 bg-indigo-600 hover:bg-indigo-500 text-white font-medium rounded-xl text-xs transition flex items-center justify-center gap-1.5"
                >
                  <Plus className="w-4 h-4" /> Approve and Add Tasks
                </button>
              </div>
            )}
          </div>
        )}

        {activeTab === "analyze" && (
          <div className="space-y-6 text-xs">
            <button
              onClick={handleAnalyzeBoard}
              disabled={loading}
              className="w-full py-2 bg-indigo-600/10 hover:bg-indigo-600 border border-indigo-500/20 hover:border-indigo-500 text-indigo-400 hover:text-white font-medium rounded-xl transition"
            >
              {loading ? "Analyzing..." : "Analyze Board Health"}
            </button>

            {analysis && (
              <div className="space-y-5">
                {/* Health & Risk score */}
                <div className="grid grid-cols-2 gap-4">
                  <div className="p-4 border border-slate-900 bg-slate-900/10 rounded-xl text-center">
                    <div className="text-2xl font-extrabold text-indigo-400">{analysis.health_score}</div>
                    <div className="text-[9px] uppercase text-slate-500 mt-1">Health Score</div>
                  </div>
                  <div className="p-4 border border-slate-900 bg-slate-900/10 rounded-xl text-center">
                    <div className="text-2xl font-extrabold text-rose-400 uppercase">{analysis.risk_level}</div>
                    <div className="text-[9px] uppercase text-slate-500 mt-1">Risk Level</div>
                  </div>
                </div>

                {/* Insights */}
                <div className="space-y-2">
                  <h5 className="font-semibold text-slate-300 flex items-center gap-1 text-[11px]">
                    <AlertTriangle className="w-3.5 h-3.5 text-amber-500" /> Key Insights
                  </h5>
                  <ul className="list-disc pl-4 space-y-1 text-slate-400 font-light leading-relaxed">
                    {analysis.insights?.map((ins: string, idx: number) => (
                      <li key={idx}>{ins}</li>
                    ))}
                  </ul>
                </div>

                {/* Recommendations */}
                <div className="space-y-2 pt-2 border-t border-slate-900/60">
                  <h5 className="font-semibold text-slate-300 flex items-center gap-1 text-[11px]">
                    <CheckSquare className="w-3.5 h-3.5 text-emerald-500" /> Recommendations
                  </h5>
                  <ul className="list-disc pl-4 space-y-1 text-slate-400 font-light leading-relaxed">
                    {analysis.recommendations?.map((rec: string, idx: number) => (
                      <li key={idx}>{rec}</li>
                    ))}
                  </ul>
                </div>
              </div>
            )}
          </div>
        )}

        {activeTab === "sprint" && (
          <div className="space-y-6 text-xs">
            <button
              onClick={handleSprintRec}
              disabled={loading}
              className="w-full py-2 bg-indigo-600/10 hover:bg-indigo-600 border border-indigo-500/20 hover:border-indigo-500 text-indigo-400 hover:text-white font-medium rounded-xl transition"
            >
              {loading ? "Recommending..." : "Get Next Sprint Recommendation"}
            </button>

            {sprintRec && (
              <div className="space-y-4">
                <div className="p-4 border border-slate-900 bg-slate-900/10 rounded-xl flex justify-between items-center">
                  <div>
                    <span className="text-[9px] uppercase text-slate-500">Recommended story points:</span>
                    <p className="text-xl font-bold text-white mt-0.5">{sprintRec.total_story_points} pts</p>
                  </div>
                  <span className="text-[9px] uppercase bg-emerald-500/10 border border-emerald-500/20 text-emerald-400 px-2 py-0.5 rounded font-bold">
                    Safe capacity
                  </span>
                </div>

                <div className="space-y-2">
                  <h5 className="font-semibold text-slate-300">Recommended Backlog Items</h5>
                  <div className="space-y-2">
                    {sprintRec.recommended?.map((rec: any, idx: number) => (
                      <div key={idx} className="p-2.5 rounded-lg border border-slate-900 bg-slate-900/20 flex justify-between items-center gap-3">
                        <span className="text-slate-300 truncate font-light">{rec.title}</span>
                        <span className="text-[9px] bg-slate-900 border border-slate-850 px-1.5 py-0.5 rounded text-slate-400">
                          {rec.story_points} pts
                        </span>
                      </div>
                    ))}
                  </div>
                </div>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
}
