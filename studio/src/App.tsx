import { BrowserRouter, Routes, Route } from 'react-router-dom';
import { Layout } from './components/Layout';
import {
  Overview,
  Traces,
  TraceDetail,
  Metrics,
  Events,
  Sessions,
  SessionDetail,
  Agents,
  AgentDetail,
  Settings,
  Evaluations,
  Benchmarks,
  AgentTopology,
} from './pages';

function App() {
  return (
    <BrowserRouter basename="/studio">
      <Routes>
        <Route path="/" element={<Layout />}>
          <Route index element={<Overview />} />
          <Route path="traces" element={<Traces />} />
          <Route path="traces/:id" element={<TraceDetail />} />
          <Route path="metrics" element={<Metrics />} />
          <Route path="events" element={<Events />} />
          <Route path="sessions" element={<Sessions />} />
          <Route path="sessions/:id" element={<SessionDetail />} />
          <Route path="agents" element={<Agents />} />
          <Route path="agents/:id" element={<AgentDetail />} />
          <Route path="evaluations" element={<Evaluations />} />
          <Route path="benchmarks" element={<Benchmarks />} />
          <Route path="topology" element={<AgentTopology />} />
          <Route path="settings" element={<Settings />} />
        </Route>
      </Routes>
    </BrowserRouter>
  );
}

export default App;
