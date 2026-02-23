import React, { useState } from "react";
import Sidebar from "./Sidebar";
import { useFolders } from "../hooks/useFolders";
import { NewFolderModal } from "./FolderModal";

interface FileItem {
  id: string;
  name: string;
  type: "file" | "folder";
  modifiedAt: Date;
}

export const FolderItem = ({
  folder,
  onClick,
}: {
  folder: any;
  onClick: () => void;
}) => {
  return (
    <div
      onClick={onClick}
      className="group flex items-center gap-3 p-4 bg-white rounded-lg border border-gray-200 hover:border-blue-400 hover:shadow-md transition-all cursor-pointer"
    >
      <div className="p-2 bg-blue-50 rounded-lg group-hover:bg-blue-100">
        <svg
          className="w-8 h-8 text-blue-500"
          fill="currentColor"
          viewBox="0 0 20 20"
        >
          <path d="M2 6a2 2 0 012-2h5l2 2h5a2 2 0 012 2v6a2 2 0 01-2 2H4a2 2 0 01-2-2V6z" />
        </svg>
      </div>
      <div className="flex-1 min-w-0">
        <p className="text-sm font-semibold text-gray-900 truncate">
          {folder.Name}
        </p>
        <p className="text-xs text-gray-500">Folder</p>
      </div>
    </div>
  );
};

export const FileItem = ({ file }: { file: any }) => {
  return (
    <div className="flex items-center gap-3 p-4 bg-white rounded-lg border border-gray-200 hover:shadow-sm transition-shadow">
      <div className="p-2 bg-gray-50 rounded-lg">
        <svg
          className="w-8 h-8 text-gray-400"
          fill="currentColor"
          viewBox="0 0 20 20"
        >
          <path d="M4 4a2 2 0 012-2h4.586A2 2 0 0112 2.586L15.414 6A2 2 0 0116 7.414V16a2 2 0 01-2 2H6a2 2 0 01-2-2V4z" />
        </svg>
      </div>
      <div className="flex-1 min-w-0">
        <p className="text-sm font-medium text-gray-900 truncate">
          {file.Name}
        </p>
        <p className="text-xs text-gray-500">File</p>
      </div>
    </div>
  );
};

const Dashboard: React.FC = () => {
  const [items, setItems] = useState<FileItem[]>([]);

  // Navigation state: null means root
  const [currentParentId, setCurrentParentId] = useState<string | null>(null);
  const [isModalOpen, setIsModalOpen] = useState(false);

  // TanStack Query handles all the data fetching logic based on the ID
  const { folders, createFolder, isCreating, isLoading } =
    useFolders(currentParentId);

  console.log({ folders });

  const handleNewFolder = (name: string) => {
    if (isCreating) return;

    createFolder(
      { name, parentId: currentParentId },
      {
        onSuccess: () => setIsModalOpen(false), // Only close if successful
      },
    );
  };

  const handleFileUpload = (files: FileList) => {
    const newFiles: FileItem[] = Array.from(files).map((file) => ({
      id: `${Date.now()}-${file.name}`,
      name: file.name,
      type: "file",
      modifiedAt: new Date(file.lastModified),
    }));
    setItems([...items, ...newFiles]);
  };

  const handleFolderUpload = (files: FileList) => {
    // Note: This is a simplified version. In a real app, you'd want to preserve folder structure
    const newFolders: FileItem[] = Array.from(files).map((file) => ({
      id: `${Date.now()}-${file.name}`,
      name: file.name,
      type: "file", // You'd want to detect folders properly
      modifiedAt: new Date(file.lastModified),
    }));
    setItems([...items, ...newFolders]);
  };

  // Logic to handle "empty state" without ternary nesting
  const renderContent = () => {
    if (isLoading)
      return <div className="p-8 text-gray-500">Loading your files...</div>;

    if (folders.length === 0) {
      return (
        <div className="col-span-full text-center py-20 bg-gray-50 rounded-xl border-2 border-dashed border-gray-200">
          <p className="text-gray-500">This folder is empty.</p>
        </div>
      );
    }

    return (
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
        {folders.map((item) => (
          <FolderItem
            key={item.ID}
            folder={item}
            onClick={() => setCurrentParentId(item.ID)}
          />
        ))}
      </div>
    );
  };

  return (
    <div className="flex h-screen bg-white">
      <Sidebar
        onNewFolder={() => setIsModalOpen(true)}
        onFileUpload={handleFileUpload}
        onFolderUpload={handleFolderUpload}
      />

      <main className="flex-1 overflow-auto p-8">
        <header className="flex items-center justify-between mb-8">
          <div>
            <nav className="flex items-center gap-2 text-sm text-gray-500 mb-1">
              <span
                className="cursor-pointer hover:text-blue-600"
                onClick={() => setCurrentParentId(null)}
              >
                Root
              </span>
              {currentParentId && <span>/ Subfolder</span>}
            </nav>
            <h1 className="text-2xl font-bold text-gray-900">My Drive</h1>
          </div>
        </header>

        {renderContent()}
      </main>

      {/* The Modal */}
      <NewFolderModal
        isOpen={isModalOpen}
        onClose={() => setIsModalOpen(false)}
        onSubmit={handleNewFolder}
        isPending={isCreating}
      />
    </div>
  );

  // return (
  //   <div className="flex h-screen bg-gray-50">
  //     <Sidebar
  //       onNewFolder={handleNewFolder}
  //       onFileUpload={handleFileUpload}
  //       onFolderUpload={handleFolderUpload}
  //     />

  //     {/* Main content area */}
  //     <div className="flex-1 overflow-auto">
  //       <div className="p-8">
  //         <h1 className="text-2xl font-semibold text-gray-800 mb-6">My Drive</h1>

  //         {/* Items grid */}
  //         <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
  //           {items.map((item) => (
  //             <div
  //               key={item.id}
  //               className="bg-white rounded-lg shadow-sm border border-gray-200 p-4 hover:shadow-md transition-shadow"
  //             >
  //               <div className="flex items-center gap-3">
  //                 {item.type === 'folder' ? (
  //                   <svg className="w-8 h-8 text-blue-500" fill="currentColor" viewBox="0 0 20 20">
  //                     <path d="M2 6a2 2 0 012-2h5l2 2h5a2 2 0 012 2v6a2 2 0 01-2 2H4a2 2 0 01-2-2V6z" />
  //                   </svg>
  //                 ) : (
  //                   <svg className="w-8 h-8 text-gray-500" fill="currentColor" viewBox="0 0 20 20">
  //                     <path fillRule="evenodd" d="M4 4a2 2 0 012-2h4.586A2 2 0 0112 2.586L15.414 6A2 2 0 0116 7.414V16a2 2 0 01-2 2H6a2 2 0 01-2-2V4zm2 6a1 1 0 011-1h6a1 1 0 110 2H7a1 1 0 01-1-1zm1 3a1 1 0 100 2h6a1 1 0 100-2H7z" clipRule="evenodd" />
  //                   </svg>
  //                 )}
  //                 <div className="flex-1 min-w-0">
  //                   <p className="text-sm font-medium text-gray-900 truncate">
  //                     {item.name}
  //                   </p>
  //                   <p className="text-xs text-gray-500">
  //                     Modified {item.modifiedAt.toLocaleDateString()}
  //                   </p>
  //                 </div>
  //               </div>
  //             </div>
  //           ))}

  //           {items.length === 0 && (
  //             <div className="col-span-full text-center py-12">
  //               <p className="text-gray-500">No items yet. Click "New" to get started.</p>
  //             </div>
  //           )}
  //         </div>
  //       </div>
  //     </div>
  //   </div>
  // );
};

export default Dashboard;
