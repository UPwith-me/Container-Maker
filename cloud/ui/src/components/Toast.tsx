import { Toaster, toast } from 'sonner'

export { Toaster, toast }

// Re-export common toast methods with proper typing
export const showSuccess = (message: string) => toast.success(message)
export const showError = (message: string) => toast.error(message)
export const showLoading = (message: string) => toast.loading(message)
export const showInfo = (message: string) => toast.info(message)

// Promise-based toast for async operations
export const showPromise = <T,>(
    promise: Promise<T>,
    messages: { loading: string; success: string; error: string }
) => toast.promise(promise, messages)
