"use client";

import { useState } from "react";
import { Task } from "@/lib/api";
import { ChevronLeft, ChevronRight, Clock } from "lucide-react";

interface CalendarViewProps {
  tasks: Task[];
  onSelectTask: (taskId: string) => void;
}

export function CalendarView({ tasks, onSelectTask }: CalendarViewProps) {
  const [currentDate, setCurrentDate] = useState(new Date());

  const year = currentDate.getFullYear();
  const month = currentDate.getMonth();

  // Month details
  const firstDayIndex = new Date(year, month, 1).getDay();
  const totalDays = new Date(year, month + 1, 0).getDate();

  const handlePrevMonth = () => {
    setCurrentDate(new Date(year, month - 1, 1));
  };

  const handleNextMonth = () => {
    setCurrentDate(new Date(year, month + 1, 1));
  };

  // Get tasks that are due on a specific date (year, month, day)
  const getTasksForDay = (day: number) => {
    return tasks.filter((t) => {
      if (!t.due_date) return false;
      const d = new Date(t.due_date);
      return d.getFullYear() === year && d.getMonth() === month && d.getDate() === day;
    });
  };

  const daysGrid = [];
  // Add empty slots for days of week before first day of month
  for (let i = 0; i < firstDayIndex; i++) {
    daysGrid.push(null);
  }
  // Add days of the month
  for (let i = 1; i <= totalDays; i++) {
    daysGrid.push(i);
  }

  const monthNames = [
    "January", "February", "March", "April", "May", "June",
    "July", "August", "September", "October", "November", "December"
  ];

  const daysOfWeek = ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"];

  return (
    <div className="border border-slate-900 bg-slate-950/45 backdrop-blur-md rounded-2xl p-6 space-y-6">
      {/* Header Calendar controls */}
      <div className="flex items-center justify-between pb-4 border-b border-slate-900/60">
        <div>
          <h3 className="font-semibold text-slate-200 text-sm">Monthly Calendar</h3>
          <p className="text-[11px] text-slate-500 font-light mt-0.5">Track and plan due dates for sprint tasks.</p>
        </div>

        <div className="flex items-center gap-2">
          <button
            onClick={handlePrevMonth}
            className="p-2 border border-slate-900 bg-slate-900/20 hover:bg-slate-900 text-slate-400 hover:text-white rounded-lg transition"
          >
            <ChevronLeft className="w-4 h-4" />
          </button>
          <span className="text-xs font-semibold text-slate-200 w-28 text-center">
            {monthNames[month]} {year}
          </span>
          <button
            onClick={handleNextMonth}
            className="p-2 border border-slate-900 bg-slate-900/20 hover:bg-slate-900 text-slate-400 hover:text-white rounded-lg transition"
          >
            <ChevronRight className="w-4 h-4" />
          </button>
        </div>
      </div>

      {/* Week Headers */}
      <div className="grid grid-cols-7 gap-2 text-center text-[10px] uppercase font-bold text-slate-500 tracking-wider">
        {daysOfWeek.map((day) => (
          <div key={day} className="py-1">
            {day}
          </div>
        ))}
      </div>

      {/* Days Grid */}
      <div className="grid grid-cols-7 gap-2">
        {daysGrid.map((day, idx) => {
          if (day === null) {
            return <div key={`empty-${idx}`} className="aspect-square bg-transparent" />;
          }

          const dayTasks = getTasksForDay(day);
          const hasTasks = dayTasks.length > 0;

          return (
            <div
              key={`day-${day}`}
              className={`aspect-square p-2 border border-slate-900 bg-slate-900/5 hover:bg-slate-900/20 rounded-xl flex flex-col justify-between items-start transition relative group cursor-pointer ${
                hasTasks ? "ring-1 ring-indigo-500/10 hover:border-indigo-500/35" : ""
              }`}
            >
              <span className={`text-xs font-medium ${hasTasks ? "text-indigo-400 font-bold" : "text-slate-500"}`}>
                {day}
              </span>

              {/* Task Dots/Pills indicator */}
              {hasTasks && (
                <div className="w-full space-y-1 mt-1">
                  <div className="flex gap-1">
                    {dayTasks.slice(0, 3).map((t) => (
                      <span
                        key={t.id}
                        onClick={(e) => {
                          e.stopPropagation();
                          onSelectTask(t.id);
                        }}
                        className={`w-1.5 h-1.5 rounded-full ${
                          t.priority === "high" ? "bg-rose-500" : t.priority === "medium" ? "bg-amber-500" : "bg-emerald-500"
                        }`}
                      />
                    ))}
                  </div>

                  {/* Desktop Hover Task Overlay list */}
                  <div className="hidden group-hover:block absolute bottom-full left-0 z-30 w-48 p-2.5 rounded-xl border border-slate-900 bg-slate-950 shadow-xl space-y-1.5">
                    {dayTasks.map((t) => (
                      <div
                        key={t.id}
                        onClick={(e) => {
                          e.stopPropagation();
                          onSelectTask(t.id);
                        }}
                        className="p-1.5 rounded-md hover:bg-slate-900 text-[10px] text-slate-300 hover:text-white truncate font-light flex items-center gap-1"
                      >
                        <span className={`w-1.5 h-1.5 rounded-full shrink-0 ${
                          t.priority === "high" ? "bg-rose-500" : t.priority === "medium" ? "bg-amber-500" : "bg-emerald-500"
                        }`} />
                        {t.title}
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>
          );
        })}
      </div>
    </div>
  );
}
