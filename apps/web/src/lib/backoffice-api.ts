export type BackofficeErrorPayload = {
  message: string
  errors?: Record<string, string>
}

export type BackofficeRequestError = Error & {
  status: number
  payload: BackofficeErrorPayload
}

let csrfToken: string | null = null

export function setBackofficeCSRFToken(nextToken?: string | null) {
  csrfToken = nextToken ?? null
}

export function clearBackofficeCSRFToken() {
  csrfToken = null
}

export function isBackofficeRequestError(
  error: unknown,
): error is BackofficeRequestError {
  return (
    error instanceof Error &&
    "status" in error &&
    "payload" in error &&
    typeof (error as BackofficeRequestError).status === "number"
  )
}

export async function backofficeRequest<T>(
  path: string,
  init?: RequestInit,
): Promise<T> {
  const method = init?.method?.toUpperCase() ?? "GET"
  const hasBody = init?.body != null
  const isFormData = typeof FormData !== "undefined" && init?.body instanceof FormData

  const response = await fetch(path, {
    credentials: "include",
    ...init,
    headers: {
      Accept: "application/json",
      ...(hasBody && !isFormData ? { "Content-Type": "application/json" } : {}),
      ...(init?.headers ?? {}),
      ...(csrfToken && method !== "GET" && method !== "HEAD"
        ? { "X-CSRF-Token": csrfToken }
        : {}),
    },
  })

  if (!response.ok) {
    const fallback: BackofficeErrorPayload = {
      message: response.status === 401 ? "Unauthorized" : "Request failed",
    }

    let payload = fallback
    try {
      payload = (await response.json()) as BackofficeErrorPayload
    } catch {
      payload = fallback
    }

    const error = new Error(payload.message) as BackofficeRequestError
    error.status = response.status
    error.payload = payload
    throw error
  }

  return response.json() as Promise<T>
}
