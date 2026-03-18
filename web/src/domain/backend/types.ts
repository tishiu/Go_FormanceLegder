export type SessionUser = {
    id: string;
    email: string;
    organization_id: string;
    role: string;
    default_project_id?: string;
};

export type SessionSnapshot = {
    status: "unknown" | "authenticated" | "anonymous";
    user: SessionUser | null;
};

export type BackendOp =
    | "auth.login"
    | "auth.register"
    | "auth.logout"
    | "ledger.list"
    | "ledger.get"
    | "ledger.create"
    | "apiKey.list"
    | "apiKey.create"
    | "apiKey.revoke";

export interface BackendInput {
    "auth.login": { email: string; password: string };
    "auth.register": { email: string; password: string };
    "auth.logout": undefined;
    "ledger.list": undefined;
    "ledger.get": { id: string };
    "ledger.create": { project_id: string; name: string; code: string; currency: string };
    "apiKey.list": { ledger_id: string };
    "apiKey.create": { ledger_id: string; description: string };
    "apiKey.revoke": { id: string };
}

export interface BackendOutput {
    "auth.login": SessionSnapshot;
    "auth.register": SessionSnapshot;
    "auth.logout": void;
    "ledger.list": unknown[];
    "ledger.get": unknown;
    "ledger.create": unknown;
    "apiKey.list": unknown[];
    "apiKey.create": unknown;
    "apiKey.revoke": void;
}

export interface BackendPort {
    bootstrap(): Promise<SessionSnapshot>;
    request<O extends BackendOp>(op: O, input: BackendInput[O]): Promise<BackendOutput[O]>;
    onSessionChange(listener: (snapshot: SessionSnapshot) => void): () => void;
}

