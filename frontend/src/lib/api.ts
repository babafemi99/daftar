export type FieldError = {
  path: string;
  code: string;
  message: string;
};

export type ApiSuccess<T> = {
  data: T;
  requestId: string;
};

export type PageMeta = {
  number: number;
  size: number;
  totalItems: number;
  totalPages: number;
  hasMore: boolean;
};

export type Paginated<T> = { items: T[]; page: PageMeta };

export type ApiErrorBody = {
  code: string;
  message: string;
  details?: { fields?: FieldError[] };
  requestId: string;
};

export type ApiErrorEnvelope = { error: ApiErrorBody };

export type User = {
  id: string;
  email: string;
  first_name: string;
  last_name: string;
  createdAt?: string;
  updatedAt?: string;
};

export type LoginInput = { email: string; password: string };
export type RegisterInput = LoginInput & { first_name: string; last_name: string };

export type Currency = "USD" | "AED" | "SAR" | "NGN" | "GBP" | "EUR";
export type DocumentStatus = "draft" | "finalized";
export type DocumentTotals = {
  subtotal: string;
  discount: string;
  tax: string;
  grandTotal: string;
};
export type CalculatedLine = {
  subtotal: string;
  discountAmount: string;
  discountedAmount: string;
  taxAmount: string;
  lineTotal: string;
};
export type DocumentLine = LineInput & { id: string; calculated: CalculatedLine };
export type TaxBreakdown = { rate: string; taxableAmount: string; taxAmount: string };
export type Document = {
  id: string;
  reference: string;
  title: string;
  customer: string;
  issueDate: string;
  currency: Currency;
  status: DocumentStatus;
  version: number;
  lineItems: DocumentLine[];
  totals: DocumentTotals;
  taxBreakdown: TaxBreakdown[];
  calculationPolicyVersion?: string;
  finalizedAt: string | null;
  archivedAt: string | null;
  createdAt: string;
  updatedAt: string;
};
export type DiscountInput = { type: "fixed" | "percentage"; value: string };
export type LineInput = { id?: string; description: string; quantity: number; unitPrice: string; discount?: DiscountInput; taxRate: string };
export type DocumentInput = { title: string; customer: string; issueDate: string; currency: Currency; lineItems: LineInput[] };
export type CalculationPreview = {
  lineItems: Array<{ calculated: { lineTotal: string } }>;
  totals: DocumentTotals;
  taxBreakdown: Array<{ rate: string; taxableAmount: string; taxAmount: string }>;
  calculationPolicyVersion: string;
};
export type DocumentListFilters = {
	search?: string;
  status?: DocumentStatus;
  currency?: Currency;
  from?: string;
  to?: string;
  archived?: boolean;
};
export type CurrencyReport = {
  currency: Currency;
  documentCount: number;
  subtotal: string;
  totalDiscount: string;
  totalTax: string;
  grandTotal: string;
  taxBreakdown: TaxBreakdown[];
};
export type SummaryReport = {
  from: string;
  to: string;
  documentCount: number;
  currencies: CurrencyReport[];
};
export type AuditAction = "document.created" | "document.updated" | "document.finalized" | "document.duplicated" | "document.archived" | "document.restored";
export type AuditEvent = {
  id: string;
  actorId: string;
  documentId: string;
  documentReference: string;
  action: AuditAction;
  documentVersion: number;
  metadata: {
    changedFields?: string[];
    sourceDocumentId?: string;
    calculationPolicyVersion?: string;
  };
  requestId: string;
  occurredAt: string;
};

export class ApiClientError extends Error {
  constructor(
    public readonly status: number,
    public readonly code: string,
    message: string,
    public readonly fields: FieldError[] = [],
    public readonly requestId = "",
  ) {
    super(message);
    this.name = "ApiClientError";
  }
}

let refreshRequest: Promise<boolean> | null = null;

async function fetchWithSession(path: string, init: RequestInit): Promise<Response> {
	let response = await fetch(path, { ...init, credentials: "include" });
	const authEndpoint = path === "/api/v1/auth/login" || path === "/api/v1/auth/register" || path === "/api/v1/auth/refresh";
	if (response.status !== 401 || authEndpoint) return response;
	if (!refreshRequest) {
		refreshRequest = fetch("/api/v1/auth/refresh", { method: "POST", credentials: "include" })
			.then((refresh) => refresh.ok)
			.catch(() => false)
			.finally(() => { refreshRequest = null; });
	}
	if (await refreshRequest) response = await fetch(path, { ...init, credentials: "include" });
	return response;
}

export async function apiRequest<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers);
  if (init.body && !headers.has("Content-Type")) headers.set("Content-Type", "application/json");

  const response = await fetchWithSession(path, { ...init, headers });
  if (response.status === 204) return undefined as T;

  let payload: ApiSuccess<T> | ApiErrorEnvelope;
  try {
    payload = (await response.json()) as ApiSuccess<T> | ApiErrorEnvelope;
  } catch {
    throw new ApiClientError(response.status, "INVALID_RESPONSE", "Daftar returned an invalid response.");
  }

  if (!response.ok) {
    const error = "error" in payload ? payload.error : undefined;
    const apiError = new ApiClientError(
      response.status,
      error?.code ?? "REQUEST_FAILED",
      error?.message ?? "The request could not be completed.",
      error?.details?.fields ?? [],
      error?.requestId ?? "",
    );
    const isAuthAttempt = path === "/api/v1/auth/login" || path === "/api/v1/auth/register";
    if (response.status === 401 && !isAuthAttempt && typeof window !== "undefined") {
      window.dispatchEvent(new Event("daftar:session-expired"));
    }
    throw apiError;
  }
  if (!("data" in payload)) throw new ApiClientError(response.status, "INVALID_RESPONSE", "Daftar returned an invalid response.");
  return payload.data;
}

async function apiCollectionRequest<T>(path: string, init: RequestInit = {}): Promise<Paginated<T>> {
  const response = await fetchWithSession(path, init);
  const payload = await response.json() as { data?: T[]; page?: PageMeta; error?: ApiErrorBody };
  if (!response.ok) {
    throw new ApiClientError(response.status, payload.error?.code ?? "REQUEST_FAILED", payload.error?.message ?? "The request could not be completed.", payload.error?.details?.fields ?? [], payload.error?.requestId ?? "");
  }
  if (!Array.isArray(payload.data) || !payload.page) throw new ApiClientError(response.status, "INVALID_RESPONSE", "Daftar returned an invalid collection response.");
  return { items: payload.data, page: payload.page };
}

export const authApi = {
  login: (input: LoginInput) =>
    apiRequest<User>("/api/v1/auth/login", { method: "POST", body: JSON.stringify(input) }),
  register: (input: RegisterInput) =>
    apiRequest<User>("/api/v1/auth/register", { method: "POST", body: JSON.stringify(input) }),
  me: () => apiRequest<User>("/api/v1/me"),
  logout: () => apiRequest<void>("/api/v1/auth/logout", { method: "POST" }),
};

export const documentsApi = {
  list: (filters: DocumentListFilters, page = 1, pageSize = 10, signal?: AbortSignal) => {
    const query = new URLSearchParams({ limit: String(pageSize), page: String(page), sort: "issue_date_desc" });
    if (filters.status) query.set("status", filters.status);
	if (filters.search) query.set("search", filters.search);
    if (filters.currency) query.set("currency", filters.currency);
    if (filters.from) query.set("from", filters.from);
    if (filters.to) query.set("to", filters.to);
    if (filters.archived !== undefined) query.set("archived", String(filters.archived));
    return apiCollectionRequest<Document>(`/api/v1/documents?${query}`, { signal });
  },
  preview: (currency: Currency, lineItems: LineInput[], signal?: AbortSignal) =>
    apiRequest<CalculationPreview>("/api/v1/documents/preview-calculation", {
      method: "POST", body: JSON.stringify({ currency, lineItems }), signal,
    }),
  create: (input: DocumentInput) =>
    apiRequest<Document>("/api/v1/documents", { method: "POST", body: JSON.stringify(input) }),
  get: (id: string, signal?: AbortSignal) => apiRequest<Document>(`/api/v1/documents/${id}`, { signal }),
  replace: (id: string, version: number, input: DocumentInput) =>
    apiRequest<Document>(`/api/v1/documents/${id}`, { method: "PATCH", headers: { "If-Match": `"${version}"` }, body: JSON.stringify(input) }),
  finalize: (id: string, version: number) =>
    apiRequest<Document>(`/api/v1/documents/${id}/finalize`, { method: "POST", headers: { "If-Match": `"${version}"` } }),
  archive: (id: string, version: number) =>
    apiRequest<Document>(`/api/v1/documents/${id}`, { method: "DELETE", headers: { "If-Match": `"${version}"` } }),
  restore: (id: string, version: number) =>
    apiRequest<Document>(`/api/v1/documents/${id}/restore`, { method: "POST", headers: { "If-Match": `"${version}"` } }),
  duplicate: (id: string) => apiRequest<Document>(`/api/v1/documents/${id}/duplicate`, { method: "POST" }),
  auditEvents: async (id: string, signal?: AbortSignal) =>
    (await apiCollectionRequest<AuditEvent>(`/api/v1/documents/${id}/audit-events?limit=25`, { signal })).items,
};

export const reportsApi = {
  summary: (from: string, to: string, signal?: AbortSignal) => {
    const query = new URLSearchParams({ from, to });
    return apiRequest<SummaryReport>(`/api/v1/reports/summary?${query}`, { signal });
  },
};
