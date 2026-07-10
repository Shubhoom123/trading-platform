import {
  createContext,
  useCallback,
  useContext,
  useMemo,
  useState,
  type ReactNode,
} from "react";
import { api, tokens } from "../api/client";

interface AuthState {
  authed: boolean;
  email: string | null;
  login: (email: string, password: string) => Promise<void>;
  register: (email: string, password: string) => Promise<void>;
  logout: () => void;
}

const AuthContext = createContext<AuthState | null>(null);

// Best-effort decode of the "email" claim from a JWT payload — display only,
// never trusted for anything (the server verifies the signature).
function emailFromToken(token: string | null): string | null {
  if (!token) return null;
  try {
    const payload = JSON.parse(atob(token.split(".")[1]));
    return payload.email ?? null;
  } catch {
    return null;
  }
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const [email, setEmail] = useState<string | null>(() =>
    emailFromToken(tokens.access()),
  );

  const login = useCallback(async (e: string, p: string) => {
    await api.login(e, p);
    setEmail(emailFromToken(tokens.access()));
  }, []);

  const register = useCallback(async (e: string, p: string) => {
    await api.register(e, p);
    setEmail(emailFromToken(tokens.access()));
  }, []);

  const logout = useCallback(() => {
    api.logout();
    setEmail(null);
  }, []);

  const value = useMemo<AuthState>(
    () => ({ authed: email !== null, email, login, register, logout }),
    [email, login, register, logout],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthState {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be used within AuthProvider");
  return ctx;
}
