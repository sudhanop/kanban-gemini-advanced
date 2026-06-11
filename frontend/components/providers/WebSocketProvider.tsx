"use client";

import React, { createContext, useContext, useEffect, useRef, useState } from "react";
import { useAuth } from "./AuthProvider";

interface WebSocketMessage {
  type: string;
  room_id?: string;
  user_id?: string;
  user_name?: string;
  user_avatar?: string;
  payload?: any;
  timestamp?: string;
}

interface WebSocketContextType {
  isConnected: boolean;
  joinRoom: (roomId: string) => void;
  leaveRoom: (roomId: string) => void;
  sendMessage: (type: string, roomId: string, data?: any) => void;
  registerListener: (type: string, callback: (msg: WebSocketMessage) => void) => () => void;
  presence: Array<{ user_id: string; name: string; avatar?: string }>;
}

const WebSocketContext = createContext<WebSocketContextType | undefined>(undefined);

export function WebSocketProvider({ children }: { children: React.ReactNode }) {
  const { user } = useAuth();
  const [isConnected, setIsConnected] = useState(false);
  const [presence, setPresence] = useState<Array<{ user_id: string; name: string; avatar?: string }>>([]);
  const socketRef = useRef<WebSocket | null>(null);
  const listenersRef = useRef<Record<string, Set<(msg: WebSocketMessage) => void>>>({});
  const activeRoomsRef = useRef<Set<string>>(new Set());

  // Helper to register listeners
  const registerListener = (type: string, callback: (msg: WebSocketMessage) => void) => {
    if (!listenersRef.current[type]) {
      listenersRef.current[type] = new Set();
    }
    listenersRef.current[type].add(callback);
    return () => {
      listenersRef.current[type]?.delete(callback);
    };
  };

  // Helper to send messages
  const sendMessage = (type: string, roomId: string, data?: any) => {
    if (socketRef.current && socketRef.current.readyState === WebSocket.OPEN) {
      socketRef.current.send(
        JSON.stringify({
          type,
          room_id: roomId,
          data,
        })
      );
    }
  };

  const joinRoom = (roomId: string) => {
    activeRoomsRef.current.add(roomId);
    sendMessage("join_room", roomId);
  };

  const leaveRoom = (roomId: string) => {
    activeRoomsRef.current.delete(roomId);
    sendMessage("leave_room", roomId);
  };

  useEffect(() => {
    if (!user) {
      if (socketRef.current) {
        socketRef.current.close();
      }
      setIsConnected(false);
      return;
    }

    const token = localStorage.getItem("accessToken");
    if (!token) return;

    const wsUrl = process.env.NEXT_PUBLIC_WS_URL || "ws://localhost:8080/ws";
    const socket = new WebSocket(`${wsUrl}?token=${token}`);
    socketRef.current = socket;

    socket.onopen = () => {
      setIsConnected(true);
      console.log("WebSocket connected");
      // Re-join any active rooms
      activeRoomsRef.current.forEach((roomId) => {
        socket.send(JSON.stringify({ type: "join_room", room_id: roomId }));
      });
    };

    socket.onmessage = (event) => {
      try {
        const msg: WebSocketMessage = JSON.parse(event.data);
        // Distribute to listeners
        if (listenersRef.current[msg.type]) {
          listenersRef.current[msg.type].forEach((cb) => cb(msg));
        }

        // Handle presence updates locally
        if (msg.type === "user_joined" || msg.type === "user_left" || msg.type === "presence") {
          // If the backend sends full presence status, use it
          if (msg.type === "presence" && Array.isArray(msg.payload)) {
            setPresence(msg.payload);
          } else if (msg.type === "user_joined" && msg.user_id) {
            setPresence((prev) => {
              if (prev.some((p) => p.user_id === msg.user_id)) return prev;
              return [...prev, { user_id: msg.user_id!, name: msg.user_name || "Unknown", avatar: msg.user_avatar }];
            });
          } else if (msg.type === "user_left" && msg.user_id) {
            setPresence((prev) => prev.filter((p) => p.user_id !== msg.user_id));
          }
        }
      } catch (err) {
        console.error("Error parsing WS message:", err);
      }
    };

    socket.onclose = () => {
      setIsConnected(false);
      console.log("WebSocket disconnected");
      // Reconnect after 3 seconds if user is still logged in
      setTimeout(() => {
        if (localStorage.getItem("accessToken")) {
          // Trigger effect rerun by reconnecting
          setIsConnected(false);
        }
      }, 3000);
    };

    socket.onerror = (err) => {
      console.error("WebSocket error:", err);
    };

    return () => {
      socket.close();
      socketRef.current = null;
    };
  }, [user]);

  return (
    <WebSocketContext.Provider
      value={{
        isConnected,
        joinRoom,
        leaveRoom,
        sendMessage,
        registerListener,
        presence,
      }}
    >
      {children}
    </WebSocketContext.Provider>
  );
}

export function useWebSocket() {
  const context = useContext(WebSocketContext);
  if (context === undefined) {
    throw new Error("useWebSocket must be used within a WebSocketProvider");
  }
  return context;
}
