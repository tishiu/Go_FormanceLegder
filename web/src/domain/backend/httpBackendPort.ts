import { api } from "../../api/client";
import type { BackendInput, BackendOp, BackendOutput, BackendPort, SessionSnapshot, SessionUser } from "./types";

function anonymousSession(): SessionSnapshot {
    return { status: "anonymous", user: null };
}

function authenticatedSession(user: SessionUser): SessionSnapshot {
    return { status: "authenticated", user };
}

export class HttpBackendPort implements BackendPort {
    private listeners = new Set<(snapshot: SessionSnapshot) => void>();
    private snapshot: SessionSnapshot = { status: "unknown", user: null };

    async bootstrap(): Promise<SessionSnapshot> {
        try {
            const response = await api.get<SessionUser>("/auth/me");
            this.snapshot = authenticatedSession(response.data);
            this.notify();
            return this.snapshot;
        } catch {
            this.snapshot = anonymousSession();
            this.notify();
            return this.snapshot;
        }
    }

    async request<O extends BackendOp>(op: O, input: BackendInput[O]): Promise<BackendOutput[O]> {
        try {
            switch (op) {
                case "auth.login": {
                    const response = await api.post<{ user: SessionUser }>("/auth/login", input);
                    this.snapshot = authenticatedSession(response.data.user);
                    this.notify();
                    return this.snapshot as BackendOutput[O];
                }
                case "auth.register": {
                    const response = await api.post<{ user: SessionUser }>("/auth/register", input);
                    this.snapshot = authenticatedSession(response.data.user);
                    this.notify();
                    return this.snapshot as BackendOutput[O];
                }
                case "auth.logout": {
                    document.cookie = "session=; expires=Thu, 01 Jan 1970 00:00:00 UTC; path=/;";
                    this.snapshot = anonymousSession();
                    this.notify();
                    return undefined as BackendOutput[O];
                }
                case "ledger.list": {
                    const response = await api.get<unknown[]>("/ledgers");
                    return response.data as BackendOutput[O];
                }
                case "ledger.get": {
                    const response = await api.get<unknown>(`/ledgers?id=${encodeURIComponent((input as BackendInput["ledger.get"]).id)}`);
                    return response.data as BackendOutput[O];
                }
                case "ledger.create": {
                    const response = await api.post<unknown>("/ledgers", input);
                    return response.data as BackendOutput[O];
                }
                case "apiKey.list": {
                    const ledgerID = (input as BackendInput["apiKey.list"]).ledger_id;
                    const response = await api.get<unknown[]>(`/ledgers/api-keys?ledger_id=${encodeURIComponent(ledgerID)}`);
                    return response.data as BackendOutput[O];
                }
                case "apiKey.create": {
                    const payload = input as BackendInput["apiKey.create"];
                    const response = await api.post<unknown>(`/ledgers/api-keys?ledger_id=${encodeURIComponent(payload.ledger_id)}`, {
                        description: payload.description,
                    });
                    return response.data as BackendOutput[O];
                }
                case "apiKey.revoke": {
                    const keyID = (input as BackendInput["apiKey.revoke"]).id;
                    await api.post(`/api-keys/revoke?id=${encodeURIComponent(keyID)}`);
                    return undefined as BackendOutput[O];
                }
                default: {
                    // Exhaustiveness check for BackendOp.
                    const neverOp: never = op;
                    throw new Error(`unsupported backend operation: ${neverOp}`);
                }
            }
        } catch (err: unknown) {
            const status = (err as { response?: { status?: number } })?.response?.status;
            if (status === 401) {
                this.snapshot = anonymousSession();
                this.notify();
            }
            throw err;
        }
    }

    onSessionChange(listener: (snapshot: SessionSnapshot) => void): () => void {
        this.listeners.add(listener);
        return () => {
            this.listeners.delete(listener);
        };
    }

    private notify() {
        for (const listener of this.listeners) {
            listener(this.snapshot);
        }
    }
}

export const backendPort: BackendPort = new HttpBackendPort();

