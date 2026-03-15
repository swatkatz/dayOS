import { ApolloClient, ApolloLink, HttpLink, InMemoryCache } from '@apollo/client'
import { SetContextLink } from '@apollo/client/link/context'
import { onError } from '@apollo/client/link/error'
import { ServerError } from '@apollo/client/errors'

const TOKEN_KEY = 'dayos_token'

export function getToken(): string | null {
  return localStorage.getItem(TOKEN_KEY)
}

export function setToken(token: string): void {
  localStorage.setItem(TOKEN_KEY, token)
}

export function clearToken(): void {
  localStorage.removeItem(TOKEN_KEY)
}

const httpLink = new HttpLink({ uri: import.meta.env.VITE_GRAPHQL_URL || '/graphql' })

const authLink = new SetContextLink(({ headers }) => {
  const token = getToken()
  return {
    headers: {
      ...headers,
      ...(token ? { authorization: `Bearer ${token}` } : {}),
      'X-Timezone': Intl.DateTimeFormat().resolvedOptions().timeZone,
    },
  }
})

let onUnauth: (() => void) | null = null

export function setOnUnauth(cb: () => void) {
  onUnauth = cb
}

const errorLink = onError(({ error }) => {
  if (ServerError.is(error) && error.statusCode === 401) {
    clearToken()
    onUnauth?.()
  }
})

export const client = new ApolloClient({
  link: ApolloLink.from([errorLink, authLink, httpLink]),
  cache: new InMemoryCache(),
})
