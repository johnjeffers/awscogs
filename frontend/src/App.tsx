import { Provider } from 'react-redux';
import { store } from './store';
import { Layout } from './components/layout/Layout';
import { CostDashboard } from './components/costs/CostDashboard';

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
