import { Provider } from 'react-redux';
import { CostDashboard } from './components/costs/CostDashboard';
import { Layout } from './components/layout/Layout';
import { store } from './store';

function App() {
  return (
    <Provider store={store}>
      <Layout>
        <CostDashboard />
      </Layout>
    </Provider>
  );
}

export default App;
