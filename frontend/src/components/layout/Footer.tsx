import React from 'react';
import { useAppSelector } from '../../hooks/useAppDispatch';

export const Footer: React.FC = () => {
  const config = useAppSelector((state) => state.costs.config);

  if (!config?.version) {
    return null;
  }

  const { version, gitCommit } = config.version;
  const shortCommit = gitCommit !== 'unknown' ? gitCommit.substring(0, 7) : '';

  return (
    <footer className="bg-white border-t border-gray-200 py-3">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <p className="text-sm text-gray-500 text-center">
          {version}
          {shortCommit && ` (${shortCommit})`}
        </p>
      </div>
    </footer>
  );
};
