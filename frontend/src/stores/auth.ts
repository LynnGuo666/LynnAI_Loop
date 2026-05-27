import { create } from "zustand";

interface AuthState {
  token: string;
  setToken: (token: string) => void;
  logout: () => void;
}

export const useAuthStore = create<AuthState>((set) => ({
  token: localStorage.getItem("loop_token") || "",
  setToken: (token) => {
    localStorage.setItem("loop_token", token);
    set({ token });
  },
  logout: () => {
    localStorage.removeItem("loop_token");
    set({ token: "" });
  },
}));
