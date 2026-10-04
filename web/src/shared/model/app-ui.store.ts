import { create } from 'zustand';

type AppUiState = {
	sidebarOpen: boolean;
	setSidebarOpen: (open: boolean) => void;
};

/**
 * Стор, который управляет UI-состоянием оболочки приложения.
 */
export const useAppUiStore = create<AppUiState>((set) => ({
	sidebarOpen: false,
	setSidebarOpen: (open) => set({ sidebarOpen: open }),
}));
