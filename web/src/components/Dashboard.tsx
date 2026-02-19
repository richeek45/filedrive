import React, { useState } from 'react';
import Sidebar from './Sidebar';

interface FileItem {
  id: string;
  name: string;
  type: 'file' | 'folder';
  modifiedAt: Date;
}

const Dashboard: React.FC = () => {
  const [items, setItems] = useState<FileItem[]>([]);

  const handleNewFolder = () => {
    const newFolder: FileItem = {
      id: Date.now().toString(),
      name: `New Folder ${items.filter(i => i.type === 'folder').length + 1}`,
      type: 'folder',
      modifiedAt: new Date(),
    };
    setItems([...items, newFolder]);
  };

  const handleFileUpload = (files: FileList) => {
    const newFiles: FileItem[] = Array.from(files).map(file => ({
      id: `${Date.now()}-${file.name}`,
      name: file.name,
      type: 'file',
      modifiedAt: new Date(file.lastModified),
    }));
    setItems([...items, ...newFiles]);
  };

  const handleFolderUpload = (files: FileList) => {
    // Note: This is a simplified version. In a real app, you'd want to preserve folder structure
    const newFolders: FileItem[] = Array.from(files).map(file => ({
      id: `${Date.now()}-${file.name}`,
      name: file.name,
      type: 'file', // You'd want to detect folders properly
      modifiedAt: new Date(file.lastModified),
    }));
    setItems([...items, ...newFolders]);
  };

  return (
    <div className="flex h-screen bg-gray-50">
      <Sidebar
        onNewFolder={handleNewFolder}
        onFileUpload={handleFileUpload}
        onFolderUpload={handleFolderUpload}
      />
      
      {/* Main content area */}
      <div className="flex-1 overflow-auto">
        <div className="p-8">
          <h1 className="text-2xl font-semibold text-gray-800 mb-6">My Drive</h1>
          
          {/* Items grid */}
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
            {items.map((item) => (
              <div
                key={item.id}
                className="bg-white rounded-lg shadow-sm border border-gray-200 p-4 hover:shadow-md transition-shadow"
              >
                <div className="flex items-center gap-3">
                  {item.type === 'folder' ? (
                    <svg className="w-8 h-8 text-blue-500" fill="currentColor" viewBox="0 0 20 20">
                      <path d="M2 6a2 2 0 012-2h5l2 2h5a2 2 0 012 2v6a2 2 0 01-2 2H4a2 2 0 01-2-2V6z" />
                    </svg>
                  ) : (
                    <svg className="w-8 h-8 text-gray-500" fill="currentColor" viewBox="0 0 20 20">
                      <path fillRule="evenodd" d="M4 4a2 2 0 012-2h4.586A2 2 0 0112 2.586L15.414 6A2 2 0 0116 7.414V16a2 2 0 01-2 2H6a2 2 0 01-2-2V4zm2 6a1 1 0 011-1h6a1 1 0 110 2H7a1 1 0 01-1-1zm1 3a1 1 0 100 2h6a1 1 0 100-2H7z" clipRule="evenodd" />
                    </svg>
                  )}
                  <div className="flex-1 min-w-0">
                    <p className="text-sm font-medium text-gray-900 truncate">
                      {item.name}
                    </p>
                    <p className="text-xs text-gray-500">
                      Modified {item.modifiedAt.toLocaleDateString()}
                    </p>
                  </div>
                </div>
              </div>
            ))}
            
            {items.length === 0 && (
              <div className="col-span-full text-center py-12">
                <p className="text-gray-500">No items yet. Click "New" to get started.</p>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
};

export default Dashboard;