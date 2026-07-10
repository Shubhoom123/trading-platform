import { LoginPage } from "./components/LoginPage";
import { TradingView } from "./components/TradingView";
import { useAuth } from "./auth/AuthContext";

export default function App() {
  const { authed } = useAuth();
  return authed ? <TradingView /> : <LoginPage />;
}
