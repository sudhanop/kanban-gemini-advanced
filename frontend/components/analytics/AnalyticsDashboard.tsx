"use client";

import { Task, Column } from "@/lib/api";
import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  PieChart,
  Pie,
  Cell,
  Legend,
} from "recharts";
import { BarChart3, PieChart as PieIcon, Users } from "lucide-react";

interface AnalyticsDashboardProps {
  tasks: Task[];
  columns: Column[];
}

export function AnalyticsDashboard({ tasks, columns }: AnalyticsDashboardProps) {
  // 1. Column distribution
  const columnData = columns.map((col) => ({
    name: col.name,
    count: tasks.filter((t) => t.column_id === col.id).length,
  }));

  // 2. Priority distribution
  const priorityCounts = { high: 0, medium: 0, low: 0 };
  tasks.forEach((t) => {
    const prio = (t.priority || "low").toLowerCase() as "high" | "medium" | "low";
    if (priorityCounts[prio] !== undefined) {
      priorityCounts[prio]++;
    } else {
      priorityCounts.low++;
    }
  });

  const priorityData = [
    { name: "High", value: priorityCounts.high, color: "#ef4444" },
    { name: "Medium", value: priorityCounts.medium, color: "#f59e0b" },
    { name: "Low", value: priorityCounts.low, color: "#10b981" },
  ].filter((d) => d.value > 0);

  // 3. User workload
  const userWorkload: Record<string, number> = {};
  tasks.forEach((t) => {
    if (t.assignees && t.assignees.length > 0) {
      t.assignees.forEach((a) => {
        const name = a.user?.name || "Unassigned";
        userWorkload[name] = (userWorkload[name] || 0) + 1;
      });
    } else {
      userWorkload["Unassigned"] = (userWorkload["Unassigned"] || 0) + 1;
    }
  });

  const workloadData = Object.entries(userWorkload).map(([name, count]) => ({
    name,
    tasks: count,
  }));

  return (
    <div className="space-y-8 max-w-full">
      {/* Upper Grid */}
      <div className="grid md:grid-cols-2 gap-8">
        {/* Task by Status Column */}
        <div className="p-6 border border-slate-900 bg-slate-950/45 backdrop-blur-md rounded-2xl space-y-4 flex flex-col">
          <h4 className="text-xs font-semibold text-slate-300 flex items-center gap-1.5 shrink-0">
            <BarChart3 className="w-4 h-4 text-indigo-400" /> Tasks by Status Column
          </h4>
          <div className="h-64 w-full flex-1">
            <ResponsiveContainer width="100%" height="100%">
              <BarChart data={columnData}>
                <CartesianGrid strokeDasharray="3 3" stroke="#0f172a" />
                <XAxis dataKey="name" stroke="#475569" fontSize={10} />
                <YAxis stroke="#475569" fontSize={10} />
                <Tooltip
                  contentStyle={{
                    backgroundColor: "#020617",
                    border: "1px solid #1e293b",
                    borderRadius: "8px",
                    fontSize: "11px",
                  }}
                />
                <Bar dataKey="count" fill="#6366f1" radius={[4, 4, 0, 0]} />
              </BarChart>
            </ResponsiveContainer>
          </div>
        </div>

        {/* Priority breakdown */}
        <div className="p-6 border border-slate-900 bg-slate-950/45 backdrop-blur-md rounded-2xl space-y-4 flex flex-col">
          <h4 className="text-xs font-semibold text-slate-300 flex items-center gap-1.5 shrink-0">
            <PieIcon className="w-4 h-4 text-purple-400" /> Priority Distribution
          </h4>
          <div className="h-64 w-full flex-1 relative flex items-center justify-center">
            {priorityData.length === 0 ? (
              <p className="text-xs text-slate-500 font-light">No task data to analyze.</p>
            ) : (
              <ResponsiveContainer width="100%" height="100%">
                <PieChart>
                  <Pie
                    data={priorityData}
                    cx="50%"
                    cy="50%"
                    innerRadius={60}
                    outerRadius={80}
                    paddingAngle={5}
                    dataKey="value"
                  >
                    {priorityData.map((entry, index) => (
                      <Cell key={`cell-${index}`} fill={entry.color} />
                    ))}
                  </Pie>
                  <Tooltip
                    contentStyle={{
                      backgroundColor: "#020617",
                      border: "1px solid #1e293b",
                      borderRadius: "8px",
                      fontSize: "11px",
                    }}
                  />
                  <Legend
                    verticalAlign="bottom"
                    height={36}
                    formatter={(value) => <span className="text-xs text-slate-400">{value}</span>}
                  />
                </PieChart>
              </ResponsiveContainer>
            )}
          </div>
        </div>
      </div>

      {/* Workload Analysis */}
      <div className="p-6 border border-slate-900 bg-slate-950/45 backdrop-blur-md rounded-2xl space-y-4 flex flex-col">
        <h4 className="text-xs font-semibold text-slate-300 flex items-center gap-1.5 shrink-0">
          <Users className="w-4 h-4 text-emerald-400" /> Workload Allocation per Member
        </h4>
        <div className="h-64 w-full flex-1">
          <ResponsiveContainer width="100%" height="100%">
            <BarChart data={workloadData} layout="vertical">
              <CartesianGrid strokeDasharray="3 3" stroke="#0f172a" />
              <XAxis type="number" stroke="#475569" fontSize={10} />
              <YAxis dataKey="name" type="category" stroke="#475569" fontSize={10} width={80} />
              <Tooltip
                contentStyle={{
                  backgroundColor: "#020617",
                  border: "1px solid #1e293b",
                  borderRadius: "8px",
                  fontSize: "11px",
                }}
              />
              <Bar dataKey="tasks" fill="#10b981" radius={[0, 4, 4, 0]} />
            </BarChart>
          </ResponsiveContainer>
        </div>
      </div>
    </div>
  );
}
