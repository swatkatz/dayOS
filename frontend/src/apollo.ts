import { ApolloClient, ApolloLink, HttpLink, InMemoryCache } from '@apollo/client'
import { SetContextLink } from '@apollo/client/link/context'
import { onError } from '@apollo/client/link/error'
import { ServerError } from '@apollo/client/errors'

// Clerk token getter — injected by App.tsx after Clerk loads
let tokenGetter: (() => Promise<string | null>) | null = null

export function setTokenGetter(fn: () => Promise<string | null>) {
  tokenGetter = fn
}

let onUnauth: (() => void) | null = null

export function setOnUnauth(cb: () => void) {
  onUnauth = cb
}

const httpLink = new HttpLink({ uri: import.meta.env.VITE_GRAPHQL_URL || '/graphql' })

const authLink = new SetContextLink(async (prevContext) => {
  const token = tokenGetter ? await tokenGetter() : null
  return {
    headers: {
      ...prevContext.headers,
      ...(token ? { authorization: `Bearer ${token}` } : {}),
      'X-Timezone': Intl.DateTimeFormat().resolvedOptions().timeZone,
    },
  }
})

const errorLink = onError(({ error }) => {
  if (ServerError.is(error) && error.statusCode === 401) {
    onUnauth?.()
  }
})

export const client = new ApolloClient({
  link: ApolloLink.from([errorLink, authLink, httpLink]),
  cache: new InMemoryCache(),
})
