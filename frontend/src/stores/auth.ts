import { create } from 'zustand';

export interface User {
  id: number;
  username: string;
  role: string;
  mustChangePassword: boolean;
}

interface AuthState {
  user: User | null;
  setUser: (user: User) => void;
  clear: () => void;
}

export const useAuthStore = create<AuthState>((set) => ({
  user: null,
  setUser: (user) => set({ user }),
  clear: () => set({ user: null }),
}));
