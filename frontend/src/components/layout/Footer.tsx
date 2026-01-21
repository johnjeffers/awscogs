import React from 'react';
import { useAppSelector } from '../../hooks/useAppDispatch';

export const Footer: React.FC = () => {
  const config = useAppSelector((state) => state.costs.config);

  const version = config?.version?.version;
  const gitCommit = config?.version?.gitCommit;
  const shortCommit = gitCommit && gitCommit !== 'unknown' ? gitCommit.substring(0, 7) : '';

  return (
    <footer className="bg-white border-t border-gray-200 py-3 shrink-0">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <p className="text-sm text-gray-500 text-center h-5">
          {version && (
            <>
              {version}
              {shortCommit && ` (${shortCommit})`}
            </>
          )}
        </p>
      </div>
    </footer>
  );
};
