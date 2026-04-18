import { AppRouterProvider } from "@/app/router"
import { queryClient } from "@/app/query-client"

function App() {
  return <AppRouterProvider queryClient={queryClient} />
}

export default App
