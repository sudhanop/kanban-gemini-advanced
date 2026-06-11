import { api } from "@/components/providers/AuthProvider";

// ─── Interfaces ─────────────────────────────────────────────────────────────
export interface Workspace {
  id: string;
  name: string;
  slug: string;
  type: string;
  owner_id: string;
  color: string;
}

export interface WorkspaceMember {
  id: string;
  workspace_id: string;
  user_id: string;
  role: string;
  joined_at: string;
  user?: {
    name: string;
    email: string;
    avatar: string;
  };
}

export interface Board {
  id: string;
  workspace_id: string;
  name: string;
  description: string;
  color: string;
  view_type: string;
  is_archived: boolean;
}

export interface Column {
  id: string;
  board_id: string;
  name: string;
  position: number;
  color: string;
  wip_limit: number;
}

export interface Task {
  id: string;
  board_id: string;
  column_id: string;
  title: string;
  description: string;
  position: number;
  priority: string;
  status: string;
  story_points: number;
  due_date?: string;
  start_date?: string;
  created_at: string;
  assignees?: TaskAssignee[];
  watchers?: TaskWatcher[];
  subtasks?: Subtask[];
  comments?: Comment[];
}

export interface TaskAssignee {
  id: string;
  task_id: string;
  user_id: string;
  user?: {
    name: string;
    avatar: string;
    email: string;
  };
}

export interface TaskWatcher {
  id: string;
  task_id: string;
  user_id: string;
}

export interface Subtask {
  id: string;
  task_id: string;
  title: string;
  is_completed: boolean;
  position: number;
}

export interface Comment {
  id: string;
  task_id: string;
  user_id: string;
  content: string;
  created_at: string;
  user?: {
    name: string;
    avatar: string;
  };
  reactions?: CommentReaction[];
}

export interface CommentReaction {
  id: string;
  comment_id: string;
  user_id: string;
  emoji: string;
}

export interface Sprint {
  id: string;
  board_id: string;
  name: string;
  goal: string;
  status: string;
  start_date?: string;
  end_date?: string;
}

export interface Reminder {
  id: string;
  task_id: string;
  user_id: string;
  message: string;
  remind_at: string;
  is_recurring: boolean;
  recurrence_pattern: string;
  status: string;
}

export interface Notification {
  id: string;
  user_id: string;
  title: string;
  message: string;
  type: string;
  is_read: boolean;
  created_at: string;
}

export interface Meeting {
  id: string;
  workspace_id: string;
  title: string;
  description: string;
  platform: string;
  meeting_url: string;
  scheduled_at: string;
}

// ─── API Methods ─────────────────────────────────────────────────────────────

// Workspaces
export const workspaceApi = {
  list: () => api.get<Workspace[]>("/workspaces").then((r) => r.data),
  create: (data: { name: string; type: string; color?: string }) =>
    api.post<Workspace>("/workspaces", data).then((r) => r.data),
  get: (id: string) => api.get<Workspace>(`/workspaces/${id}`).then((r) => r.data),
  update: (id: string, data: Partial<Workspace>) =>
    api.put<Workspace>(`/workspaces/${id}`, data).then((r) => r.data),
  delete: (id: string) => api.delete(`/workspaces/${id}`).then((r) => r.data),
  getMembers: (id: string) =>
    api.get<WorkspaceMember[]>(`/workspaces/${id}/members`).then((r) => r.data),
  removeMember: (workspaceId: string, memberId: string) =>
    api.delete(`/workspaces/${workspaceId}/members/${memberId}`).then((r) => r.data),
  inviteUser: (workspaceId: string, email: string, role: string) =>
    api.post(`/workspaces/${workspaceId}/invites`, { email, role }).then((r) => r.data),
  listInvites: (workspaceId: string) =>
    api.get(`/workspaces/${workspaceId}/invites`).then((r) => r.data),
};

// Boards
export const boardApi = {
  list: (workspaceId: string) =>
    api.get<Board[]>(`/workspaces/${workspaceId}/boards`).then((r) => r.data),
  create: (workspaceId: string, data: { name: string; description?: string; color?: string }) =>
    api.post<Board>(`/workspaces/${workspaceId}/boards`, data).then((r) => r.data),
  get: (workspaceId: string, boardId: string) =>
    api.get<Board>(`/workspaces/${workspaceId}/boards/${boardId}`).then((r) => r.data),
  update: (workspaceId: string, boardId: string, data: Partial<Board>) =>
    api.put<Board>(`/workspaces/${workspaceId}/boards/${boardId}`, data).then((r) => r.data),
  delete: (workspaceId: string, boardId: string) =>
    api.delete(`/workspaces/${workspaceId}/boards/${boardId}`).then((r) => r.data),
  duplicate: (workspaceId: string, boardId: string) =>
    api.post<Board>(`/workspaces/${workspaceId}/boards/${boardId}/duplicate`).then((r) => r.data),
  archive: (workspaceId: string, boardId: string) =>
    api.post(`/workspaces/${workspaceId}/boards/${boardId}/archive`).then((r) => r.data),
  
  // Columns
  getColumns: (workspaceId: string, boardId: string) =>
    api.get<Column[]>(`/workspaces/${workspaceId}/boards/${boardId}/columns`).then((r) => r.data),
  createColumn: (workspaceId: string, boardId: string, data: { name: string; wip_limit?: number; color?: string }) =>
    api.post<Column>(`/workspaces/${workspaceId}/boards/${boardId}/columns`, data).then((r) => r.data),
  updateColumn: (workspaceId: string, boardId: string, columnId: string, data: Partial<Column>) =>
    api.put<Column>(`/workspaces/${workspaceId}/boards/${boardId}/columns/${columnId}`, data).then((r) => r.data),
  deleteColumn: (workspaceId: string, boardId: string, columnId: string) =>
    api.delete(`/workspaces/${workspaceId}/boards/${boardId}/columns/${columnId}`).then((r) => r.data),
  reorderColumns: (workspaceId: string, boardId: string, columnIds: string[]) =>
    api.post(`/workspaces/${workspaceId}/boards/${boardId}/columns/reorder`, { column_ids: columnIds }).then((r) => r.data),
};

// Tasks
export const taskApi = {
  list: (workspaceId: string, boardId: string) =>
    api.get<Task[]>(`/workspaces/${workspaceId}/boards/${boardId}/tasks`).then((r) => r.data),
  create: (workspaceId: string, boardId: string, data: Partial<Task>) =>
    api.post<Task>(`/workspaces/${workspaceId}/boards/${boardId}/tasks`, data).then((r) => r.data),
  get: (workspaceId: string, boardId: string, taskId: string) =>
    api.get<Task>(`/workspaces/${workspaceId}/boards/${boardId}/tasks/${taskId}`).then((r) => r.data),
  update: (workspaceId: string, boardId: string, taskId: string, data: Partial<Task>) =>
    api.put<Task>(`/workspaces/${workspaceId}/boards/${boardId}/tasks/${taskId}`, data).then((r) => r.data),
  delete: (workspaceId: string, boardId: string, taskId: string) =>
    api.delete(`/workspaces/${workspaceId}/boards/${boardId}/tasks/${taskId}`).then((r) => r.data),
  move: (workspaceId: string, boardId: string, taskId: string, data: { column_id: string; position: number }) =>
    api.post(`/workspaces/${workspaceId}/boards/${boardId}/tasks/${taskId}/move`, data).then((r) => r.data),

  // Assignees & Watchers
  addAssignee: (workspaceId: string, boardId: string, taskId: string, userId: string) =>
    api.post(`/workspaces/${workspaceId}/boards/${boardId}/tasks/${taskId}/assignees`, { user_id: userId }).then((r) => r.data),
  removeAssignee: (workspaceId: string, boardId: string, taskId: string, userId: string) =>
    api.delete(`/workspaces/${workspaceId}/boards/${boardId}/tasks/${taskId}/assignees/${userId}`).then((r) => r.data),
  addWatcher: (workspaceId: string, boardId: string, taskId: string, userId: string) =>
    api.post(`/workspaces/${workspaceId}/boards/${boardId}/tasks/${taskId}/watch`, { user_id: userId }).then((r) => r.data),

  // Subtasks
  getSubtasks: (workspaceId: string, boardId: string, taskId: string) =>
    api.get<Subtask[]>(`/workspaces/${workspaceId}/boards/${boardId}/tasks/${taskId}/subtasks`).then((r) => r.data),
  createSubtask: (workspaceId: string, boardId: string, taskId: string, title: string) =>
    api.post<Subtask>(`/workspaces/${workspaceId}/boards/${boardId}/tasks/${taskId}/subtasks`, { title }).then((r) => r.data),
  updateSubtask: (workspaceId: string, boardId: string, taskId: string, subtaskId: string, data: Partial<Subtask>) =>
    api.put<Subtask>(`/workspaces/${workspaceId}/boards/${boardId}/tasks/${taskId}/subtasks/${subtaskId}`, data).then((r) => r.data),

  // Comments
  addComment: (workspaceId: string, boardId: string, taskId: string, content: string) =>
    api.post<Comment>(`/workspaces/${workspaceId}/boards/${boardId}/tasks/${taskId}/comments`, { content }).then((r) => r.data),
};

// Sprints
export const sprintApi = {
  list: (workspaceId: string, boardId: string) =>
    api.get<Sprint[]>(`/workspaces/${workspaceId}/boards/${boardId}/sprints`).then((r) => r.data),
  create: (workspaceId: string, boardId: string, data: { name: string; goal?: string; start_date?: string; end_date?: string }) =>
    api.post<Sprint>(`/workspaces/${workspaceId}/boards/${boardId}/sprints`, data).then((r) => r.data),
  start: (workspaceId: string, boardId: string, sprintId: string) =>
    api.post(`/workspaces/${workspaceId}/boards/${boardId}/sprints/${sprintId}/start`).then((r) => r.data),
  complete: (workspaceId: string, boardId: string, sprintId: string) =>
    api.post(`/workspaces/${workspaceId}/boards/${boardId}/sprints/${sprintId}/complete`).then((r) => r.data),
  getBurndown: (workspaceId: string, boardId: string, sprintId: string) =>
    api.get(`/workspaces/${workspaceId}/boards/${boardId}/sprints/${sprintId}/burndown`).then((r) => r.data),
};

// Reminders
export const reminderApi = {
  list: () => api.get<Reminder[]>("/reminders").then((r) => r.data),
  create: (data: { task_id: string; message: string; remind_at: string; is_recurring?: boolean; recurrence_pattern?: string }) =>
    api.post<Reminder>("/reminders", data).then((r) => r.data),
  update: (id: string, data: Partial<Reminder>) =>
    api.put<Reminder>(`/reminders/${id}`, data).then((r) => r.data),
  pause: (id: string) => api.post(`/reminders/${id}/pause`).then((r) => r.data),
  resume: (id: string) => api.post(`/reminders/${id}/resume`).then((r) => r.data),
  delete: (id: string) => api.delete(`/reminders/${id}`).then((r) => r.data),
};

// AI Actions
export const aiApi = {
  suggestTasks: (workspaceId: string, boardId: string, prompt: string) =>
    api.post(`/workspaces/${workspaceId}/boards/${boardId}/ai/suggest-tasks`, { prompt }).then((r) => r.data),
  confirmTasks: (workspaceId: string, boardId: string, suggestionId: string, taskIndices: number[]) =>
    api.post(`/workspaces/${workspaceId}/boards/${boardId}/ai/confirm-tasks`, { suggestion_id: suggestionId, task_indices: taskIndices }).then((r) => r.data),
  analyzeBoard: (workspaceId: string, boardId: string) =>
    api.get(`/workspaces/${workspaceId}/boards/${boardId}/ai/analyze`).then((r) => r.data),
  suggestPriorities: (workspaceId: string, boardId: string) =>
    api.get(`/workspaces/${workspaceId}/boards/${boardId}/ai/priorities`).then((r) => r.data),
  sprintRecommendation: (workspaceId: string, boardId: string) =>
    api.get(`/workspaces/${workspaceId}/boards/${boardId}/ai/sprint-recommendation`).then((r) => r.data),
  predictCompletion: (workspaceId: string, boardId: string, taskId: string) =>
    api.get(`/workspaces/${workspaceId}/boards/${boardId}/tasks/${taskId}/ai/predict`).then((r) => r.data),
};

// Search & Notifications
export const generalApi = {
  search: (query: string) => api.get(`/search?q=${encodeURIComponent(query)}`).then((r) => r.data),
  notifications: () => api.get<Notification[]>("/notifications").then((r) => r.data),
  markNotificationRead: (id: string) => api.patch(`/notifications/${id}/read`).then((r) => r.data),
  markAllNotificationsRead: () => api.post("/notifications/mark-all-read").then((r) => r.data),
  createMeeting: (data: { title: string; description?: string; scheduled_at: string; workspace_id: string }) =>
    api.post<Meeting>("/meetings", data).then((r) => r.data),
  getAnalytics: (workspaceId: string) =>
    api.get(`/workspaces/${workspaceId}/analytics`).then((r) => r.data),
};
