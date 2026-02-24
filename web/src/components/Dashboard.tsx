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

// Helper to format bytes into KB, MB, GB
const formatSize = (bytes: number) => {
  if (bytes === 0) return "0 Bytes";
  const k = 1024;
  const sizes = ["Bytes", "KB", "MB", "GB", "TB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + " " + sizes[i];
};

// Simple icon mapper based on MimeType
const getFileIcon = (mimeType: string | null) => {
  if (!mimeType) return "📄";
  if (mimeType.includes("image")) return "🖼️";
  if (mimeType.includes("pdf")) return "📕";
  if (mimeType.includes("video")) return "🎬";
  if (mimeType.includes("zip") || mimeType.includes("archive")) return "📦";
  return "📄";
};

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
  const isUploading =
    file.uploadStatus === "pending" || file.uploadStatus === "uploading";

  const progress =
    file.totalChunks > 0
      ? Math.round((file.uploadedChunks / file.totalChunks) * 100)
      : 0;

  return (
    <div className="group flex items-center gap-3 p-4 bg-white rounded-lg border border-gray-200 hover:border-blue-300 hover:shadow-md transition-all cursor-pointer">
      <div className="flex items-center justify-center w-12 h-12 bg-gray-50 rounded-lg text-2xl group-hover:bg-blue-50 transition-colors">
        {getFileIcon(file.mimeType)}
      </div>

      <div className="flex-1 min-w-0">
        <div className="flex items-center justify-between gap-2">
          <p className="text-sm font-semibold text-gray-900 truncate">
            {file.name}
          </p>
          {isUploading && (
            <span className="text-[10px] px-1.5 py-0.5 rounded-full bg-blue-100 text-blue-700 font-medium animate-pulse">
              {progress}%
            </span>
          )}
        </div>

        <div className="flex items-center gap-2 mt-1">
          <p className="text-xs text-gray-500">{formatSize(file.size)}</p>
          <span className="text-gray-300">•</span>
          <p className="text-xs text-gray-500">
            {new Date(file.createdAt).toLocaleDateString()}
          </p>
        </div>

        {/* Upload Progress Bar (Only visible while uploading) */}
        {isUploading && (
          <div className="w-full bg-gray-100 rounded-full h-1 mt-2">
            <div
              className="bg-blue-600 h-1 rounded-full transition-all duration-300"
              style={{ width: `${progress}%` }}
            ></div>
          </div>
        )}
      </div>
    </div>
  );
};

const Dashboard: React.FC = () => {
  const [currentParentId, setCurrentParentId] = useState<string | null>(null);
  const [isModalOpen, setIsModalOpen] = useState(false);

  const { folders, files, createFolder, uploadFile, isCreating, isLoading } =
    useFolders(currentParentId);

  console.log({ folders, files });

  const handleNewFolder = (name: string) => {
    if (isCreating) return;

    createFolder(
      { name, parentId: currentParentId },
      {
        onSuccess: () => setIsModalOpen(false),
      },
    );
  };

  const handleFileUpload = async (files: FileList) => {
    const fileArray = Array.from(files);

    // Using mutateAsync in a loop allows you to await each sequential upload
    for (const file of fileArray) {
      try {
        console.log("Initiating upload");
        await uploadFile(file);
      } catch (err) {
        console.error("Upload failed for one file, continuing with others...");
      }
    }
  };

  const handleFolderUpload = async (files: FileList) => {
    const fileArray = Array.from(files);
    for (const file of fileArray) {
      // Your backend can use file.webkitRelativePath to reconstruct folders
      await uploadFile(file);
    }
  };

  const renderContent = () => {
    if (isLoading)
      return <div className="p-8 text-gray-500">Loading your files...</div>;

    if (folders.length === 0 && files.length == 0) {
      return (
        <div className="col-span-full text-center py-20 bg-gray-50 rounded-xl border-2 border-dashed border-gray-200">
          <p className="text-gray-500">This drive is empty.</p>
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
        {files.map((item) => (
          <FileItem key={item.ID} file={item} />
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

      <NewFolderModal
        isOpen={isModalOpen}
        onClose={() => setIsModalOpen(false)}
        onSubmit={handleNewFolder}
        isPending={isCreating}
      />
    </div>
  );
};

export default Dashboard;
