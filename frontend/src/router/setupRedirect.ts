export function resolveAuthenticatedHomePath(isAdmin: boolean, isObserver = false): string {
  if (isAdmin) return '/admin/dashboard'
  return isObserver ? '/admin/accounts' : '/available-channels'
}

export function resolveCompletedSetupRedirectPath(
  isAuthenticated: boolean,
  isAdmin: boolean,
  isObserver = false
): string {
  if (!isAuthenticated) {
    return '/login'
  }

  return resolveAuthenticatedHomePath(isAdmin, isObserver)
}
