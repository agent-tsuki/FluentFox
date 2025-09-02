// App.tsx
import { BrowserRouter as Router, Routes, Route } from 'react-router-dom';
import Navbar from './components/layout/Navbar';
import Footer from './components/layout/Footer';
import Home from './pages/Home';
import AlphabetChart from './pages/Alphabet/AlphabetChart';
import './styles/index.css';

function App() {
  return (
    <Router>
      <div className="App">
        <Navbar />
        <Routes>
          <Route path="/" element={<Home />} />
          <Route path="/alphabet" element={<AlphabetChart />} />
          {/* Add other routes as needed */}
        </Routes>
        <Footer />
      </div>
    </Router>
  );
}

export default App;
