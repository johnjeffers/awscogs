import React, { useState } from 'react';
import { Footer } from './Footer';

interface LayoutProps {
  children: React.ReactNode;
}

export const Layout: React.FC<LayoutProps> = ({ children }) => {
  const [expanded, setExpanded] = useState(false);
  const containerClass = expanded ? 'mx-auto px-4 sm:px-6 lg:px-8' : 'max-w-7xl mx-auto px-4 sm:px-6 lg:px-8';

  return (
    <div className="min-h-screen bg-gray-100 flex flex-col">
      <header className="bg-white shadow shrink-0">
        <div className={`${containerClass} py-4 flex items-center justify-between`}>
          <h1 className="text-2xl font-bold text-gray-900">awsCOGS</h1>
          <button
            onClick={() => setExpanded(!expanded)}
            className="px-2 py-1 text-lg font-bold text-gray-400 hover:text-gray-600 hover:bg-gray-100 rounded font-mono tracking-widest"
            title={expanded ? 'Collapse' : 'Expand'}
          >
            {expanded ? '›‹' : '‹›'}
          </button>
        </div>
      </header>
      <div className="flex-1">
        <main className={`${containerClass} py-6`}>{children}</main>
      </div>
      <Footer expanded={expanded} />
    </div>
  );
};
