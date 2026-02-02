import { create } from 'zustand'
import { persist } from 'zustand/middleware'

export const useAuthStore = create(
  persist(
    (set, get) => ({
      token: null,
      user: null,
      
      login: (token, user) => {
        set({ token, user })
      },
      
      logout: () => {
        set({ token: null, user: null })
      },
      
      isAuthenticated: () => {
        return !!get().token
      },
    }),
    {
      name: 'auth-storage',
    }
  )
)
