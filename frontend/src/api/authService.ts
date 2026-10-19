import api from "../services/api";

export const authService = {
  /** Changes the password of the person logged in. The current one is checked by the backend. */
  async changePassword(currentPassword: string, newPassword: string): Promise<void> {
    await api.put("/auth/password", { current_password: currentPassword, new_password: newPassword });
  },
};
