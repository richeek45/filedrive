import React, { useState } from "react";
import Sidebar from "./Sidebar";
import { useFolders } from "../hooks/useFolders";
import { NewFolderModal } from "./FolderModal";
import {
  MoreVertical,
  Play,
  Download,
  Trash,
  Share,
  Edit2,
} from "lucide-react";
import { useUploadManager } from "../lib/uploadManager";
import { useLocation, useNavigate, useParams } from "react-router-dom";
import { useAuth } from "../context/AuthContext";

interface FileItem {
  id: string;
  name: string;
  type: "file" | "folder";
  modifiedAt: Date;
}

const formatSize = (bytes: number) => {
  if (bytes === 0) return "0 Bytes";
  const k = 1024;
  const sizes = ["Bytes", "KB", "MB", "GB", "TB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + " " + sizes[i];
};

const getFileIcon = (mimeType: string | null) => {
  if (!mimeType) return "📄";
  if (mimeType.includes("image")) return "🖼️";
  if (mimeType.includes("pdf")) return "📕";
  if (mimeType.includes("video")) return "🎬";
  if (mimeType.includes("zip") || mimeType.includes("archive")) return "📦";
  return "📄";
};

const ShareModal = ({ isOpen, onClose, shareFile, itemName, file }: any) => {
  const [email, setEmail] = useState("");

  if (!isOpen) return null;

  const handleShareFile = async () => {
    await shareFile({
      fileId: file.id,
      folderId: file.parentId,
      permission: "viewer",
      emails: [email],
    });
    onClose();
  };

  return (
    <div className="fixed inset-0 z-[100] flex items-center justify-center bg-black/50 backdrop-blur-sm">
      <div className="bg-white rounded-xl shadow-2xl w-full max-w-md p-6 animate-in fade-in zoom-in duration-200">
        <h2 className="text-xl font-semibold mb-1">Share "{itemName}"</h2>
        <p className="text-sm text-gray-500 mb-6">
          Add people with their email addresses.
        </p>

        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Email address
            </label>
            <input
              type="email"
              placeholder="alex@example.com"
              className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500 outline-none transition-all"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
            />
          </div>

          <div className="flex justify-end gap-3 mt-8">
            <button
              onClick={onClose}
              className="px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-100 rounded-lg transition-colors"
            >
              Cancel
            </button>
            <button
              onClick={handleShareFile}
              className="px-4 py-2 text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 rounded-lg transition-colors shadow-sm"
            >
              Send Invite
            </button>
          </div>
        </div>
      </div>
    </div>
  );
};

const RenameModal = ({ isOpen, onClose, currentName, onRename }: any) => {
  const [newName, setNewName] = useState(currentName);

  if (!isOpen) return null;

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (newName.trim() && newName !== currentName) {
      onRename(newName);
    }
    onClose();
  };

  return (
    <div className="fixed inset-0 z-[110] flex items-center justify-center bg-black/40 backdrop-blur-[2px]">
      <div className="bg-white rounded-xl shadow-2xl w-full max-w-sm p-6 animate-in zoom-in duration-150">
        <h2 className="text-lg font-semibold mb-4">Rename</h2>

        <form onSubmit={handleSubmit}>
          <input
            autoFocus
            type="text"
            className="w-full px-3 py-2 border-2 border-blue-500 rounded-lg outline-none mb-6 text-sm"
            value={newName}
            onChange={(e) => setNewName(e.target.value)}
            onFocus={(e) => e.target.select()}
          />

          <div className="flex justify-end gap-3">
            <button
              type="button"
              onClick={onClose}
              className="px-4 py-2 text-sm font-medium text-gray-600 hover:bg-gray-100 rounded-lg transition-colors"
            >
              Cancel
            </button>
            <button
              type="submit"
              className="px-4 py-2 text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 rounded-lg transition-colors"
            >
              OK
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};

export const FolderItem = ({
  folder,
  onClick,
  renameFolder,
  shareFile,
}: {
  folder: any;
  onClick: () => void;
  renameFolder: ({ id, name }: { id: string; name: string }) => void;
  shareFile: ({
    fileId,
    folderId,
    permission,
    emails,
  }: {
    fileId: string;
    folderId: string;
    emails: string[];
    permission: string;
  }) => void;
}) => {
  const [showMenu, setShowMenu] = useState(false);
  const [isShareOpen, setIsShareOpen] = useState(false);
  const [isRenameOpen, setIsRenameOpen] = useState(false);

  const handleRenameSubmit = (newName: string) => {
    console.log(`Renaming ${folder.id} to ${newName}`);
    renameFolder({ id: folder.id, name: newName });
  };

  return (
    <>
      <div className="group relative flex items-center gap-3 p-4 bg-white rounded-lg border border-gray-200 hover:border-blue-400 hover:shadow-md transition-all cursor-pointer">
        <div
          onClick={onClick}
          className="flex flex-1 items-center gap-3 min-w-0"
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
              {folder.name}
            </p>
            <p className="text-xs text-gray-500">Folder</p>
          </div>
        </div>

        {/* Action Menu */}
        <div className="relative">
          <button
            onClick={(e) => {
              e.stopPropagation();
              setShowMenu(!showMenu);
            }}
            className="p-1 text-gray-400 hover:text-gray-600 rounded-md hover:bg-gray-100"
          >
            <MoreVertical size={18} />
          </button>

          {showMenu && (
            <>
              <div
                className="fixed inset-0 z-10"
                onClick={(e) => {
                  e.stopPropagation();
                  setShowMenu(false);
                }}
              />
              <div className="absolute right-0 mt-2 w-48 bg-white border border-gray-100 rounded-lg shadow-xl z-20 py-1 overflow-hidden">
                <MenuOption
                  icon={<Share size={14} />}
                  label="Share"
                  onClick={() => setIsShareOpen(true)}
                />
                <MenuOption
                  icon={<Edit2 size={14} />}
                  label="Rename"
                  onClick={() => setIsRenameOpen(true)}
                />
                <hr className="my-1 border-gray-50" />
                {/* <MenuOption
                  icon={<Trash size={14} />}
                  label="Delete"
                  danger
                  onClick={() => deleteFolder(folder.ID)}
                /> */}
              </div>
            </>
          )}
        </div>
      </div>
      <RenameModal
        isOpen={isRenameOpen}
        onClose={() => setIsRenameOpen(false)}
        currentName={folder.name}
        onRename={handleRenameSubmit}
      />
      <ShareModal
        isOpen={isShareOpen}
        shareFile={shareFile}
        onClose={() => setIsShareOpen(false)}
        itemName={folder.name}
      />
    </>
  );
};

const MenuOption = ({ icon, label, onClick, danger = false }: any) => (
  <button
    onClick={onClick}
    className={`w-full flex items-center gap-3 px-4 py-2 text-sm transition-colors ${
      danger ? "text-red-600 hover:bg-red-50" : "text-gray-700 hover:bg-gray-50"
    }`}
  >
    {icon} {label}
  </button>
);

export const FileItem = ({
  file,
  downloadFile,
  moveToTrash,
  uploadFile,
  renameFile,
  shareFile,
}: {
  file: any;
  downloadFile: (fileId: string) => void;
  moveToTrash: (fileId: string) => void;
  uploadFile: (file: File) => void;
  renameFile: ({ id, name }: { id: string; name: string }) => void;
  shareFile: ({
    fileId,
    folderId,
    permission,
    emails,
  }: {
    fileId: string;
    folderId: string;
    emails: string[];
    permission: string;
  }) => void;
}) => {
  const [isShareOpen, setIsShareOpen] = useState(false);
  const [isRenameOpen, setIsRenameOpen] = useState(false);

  const [showMenu, setShowMenu] = useState(false);
  const isPaused = file.uploadStatus === "paused";
  const isUploading =
    file.uploadStatus === "pending" || file.uploadStatus === "uploading";

  const progress =
    file.totalChunks > 0
      ? Math.round((file.uploadedChunks / file.totalChunks) * 100)
      : 0;

  const { getPersistentFile, registerFileHandle } = useUploadManager();

  const handleResume = async (fileData: any) => {
    // Try to get the file from IndexedDB first
    let file = await getPersistentFile(fileData.id);

    // Fallback: If handle is missing, ask user to re-select
    if (!file) {
      const [handle] = await window.showOpenFilePicker({
        multiple: false,
        types: [
          { description: "Original File", accept: { [fileData.mimeType]: [] } },
        ],
      });

      file = await handle.getFile();

      // Check if it's the right file by name/size
      if (file.name !== fileData.name) {
        alert("Please select the original file: " + fileData.name);
        return;
      }

      // Re-save handle for next time
      await registerFileHandle(fileData.id, handle, fileData.mimeType);
    }
    if (file) {
      await uploadFile(
        new File([file], file.name, { type: fileData.mimeType }) as File,
      );
    }
  };

  const handleRenameSubmit = (newName: string) => {
    console.log(`Renaming ${file.id} to ${newName}`);
    renameFile({ name: newName, id: file.id });
  };

  return (
    <div className="relative group flex items-center gap-3 p-4 bg-white rounded-xl border border-gray-200 hover:border-blue-400 transition-all">
      {/* Icon Section */}
      <div className="p-2 bg-blue-50 rounded-lg text-blue-600">
        {getFileIcon(file.mimeType)}
      </div>

      <div className="flex-1 min-w-0">
        <p className="text-sm font-semibold truncate">{file.name}</p>
        {/* <p className="text-xs text-gray-500">{file.uploadStatus}</p> */}
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
        {isUploading && (
          <span className="text-[10px] px-1.5 py-0.5 rounded-full bg-blue-100 text-blue-700 font-medium animate-pulse">
            {progress}%
          </span>
        )}
      </div>

      {/* Action Area */}
      <div className="flex items-center gap-2">
        {isPaused && (
          <button
            onClick={() => {
              handleResume(file);
            }}
            className="p-1.5 bg-blue-600 text-white rounded-full hover:bg-blue-700 transition-colors"
            title="Resume Upload"
          >
            <Play size={14} fill="currentColor" />
          </button>
        )}
        {/* Menu Toggle */}
        <div className="relative">
          <button
            onClick={() => setShowMenu(!showMenu)}
            className="p-1 text-gray-400 hover:text-gray-600 rounded-md hover:bg-gray-100"
          >
            <MoreVertical size={20} />
          </button>

          {showMenu && (
            <>
              <div
                className="fixed inset-0 z-10"
                onClick={() => setShowMenu(false)}
              />
              <div className="absolute right-0 mt-2 w-48 bg-white border border-gray-100 rounded-lg shadow-xl z-20 py-1 overflow-hidden">
                <MenuOption
                  icon={<Download size={16} />}
                  label="Download"
                  onClick={() => downloadFile(file.id)}
                />
                <MenuOption
                  icon={<Edit2 size={16} />}
                  label="Rename"
                  onClick={() => setIsRenameOpen(true)}
                />
                <MenuOption
                  icon={<Share size={16} />}
                  label="Share"
                  onClick={() => {
                    setShowMenu(false);
                    setIsShareOpen(true);
                  }}
                />
                <hr className="my-1 border-gray-50" />
                <MenuOption
                  icon={<Trash size={16} />}
                  label="Move to Trash"
                  danger
                  onClick={() => moveToTrash(file.id)}
                />
              </div>
            </>
          )}
        </div>
      </div>
      <RenameModal
        isOpen={isRenameOpen}
        onClose={() => setIsRenameOpen(false)}
        currentName={file.name}
        onRename={handleRenameSubmit}
      />
      <ShareModal
        file={file}
        shareFile={shareFile}
        isOpen={isShareOpen}
        onClose={() => setIsShareOpen(false)}
        itemName={file.name}
      />
    </div>
  );
};

const Dashboard: React.FC = () => {
  const { folderId } = useParams<{ folderId?: string }>();
  const [isModalOpen, setIsModalOpen] = useState(false);
  const navigate = useNavigate();
  const { pathname } = useLocation();
  const isTrashView = pathname.includes("/trash");
  const sharedWithMeView = pathname.includes("/shared");
  const [isProfileOpen, setIsProfileOpen] = useState(false);

  const { user } = useAuth();

  const {
    folders,
    files,
    downloadFile,
    moveToTrash,
    createFolder,
    uploadFile,
    renameFile,
    renameFolder,
    shareFile,
    isCreating,
    isLoading,
    activeUploads,
  } = useFolders(folderId || null, isTrashView, sharedWithMeView);

  const handleNewFolder = (name: string) => {
    if (isCreating) return;

    createFolder(
      { name, parentId: folderId },
      {
        onSuccess: () => setIsModalOpen(false),
      },
    );
  };

  const handleFileUpload = async (files: FileList) => {
    const fileArray = Array.from(files);

    // Using mutateAsync in a loop allows you to await each sequential upload
    // Find a way to fire multiple file uploads simultaneously
    for (const file of fileArray) {
      try {
        await uploadFile(file);
      } catch (err) {
        console.error("Upload failed for one file, continuing with others...");
      }
    }
  };

  const handleFolderUpload = async (files: FileList) => {
    const fileArray = Array.from(files);
    for (const file of fileArray) {
      await uploadFile(file);
    }
  };

  const renderContent = () => {
    const emptyText = folderId
      ? "This folder is empty."
      : "This drive is empty.";
    if (isLoading)
      return <div className="p-8 text-gray-500">Loading your files...</div>;

    if (folders.length === 0 && files.length == 0) {
      return (
        <div className="col-span-full text-center py-20 bg-gray-50 rounded-xl border-2 border-dashed border-gray-200">
          <p className="text-gray-500">{emptyText}</p>
        </div>
      );
    }

    return (
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
        {folders.map((item) => (
          <FolderItem
            key={item.id}
            folder={item}
            renameFolder={renameFolder}
            shareFile={shareFile}
            onClick={() => navigate(`/dashboard/${item.id}`)}
          />
        ))}
        {files.map((item) => (
          <FileItem
            key={item.ID}
            file={item}
            renameFile={renameFile}
            downloadFile={downloadFile}
            moveToTrash={moveToTrash}
            uploadFile={uploadFile}
            shareFile={shareFile}
          />
        ))}
        {Object.entries(activeUploads).map(([fileName, progress]) => (
          <div key={fileName} className="p-2 border rounded mb-2 bg-blue-50">
            <div className="flex justify-between text-sm mb-1">
              <span>{fileName}</span>
              <span>{progress}%</span>
            </div>
            <div className="w-full bg-gray-200 h-2 rounded overflow-hidden">
              <div
                className="bg-blue-600 h-full transition-all duration-300"
                style={{ width: `${progress}%` }}
              />
            </div>
          </div>
        ))}
      </div>
    );
  };

  if (!user) return null;

  return (
    <div className="flex w-[95%] h-screen bg-white">
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
                onClick={() => navigate("/dashboard")}
              >
                Root
              </span>
              {folderId && (
                <>
                  <span>/</span>
                  <span className="font-medium text-gray-900">Subfolder</span>
                </>
              )}
            </nav>
            <h1 className="text-2xl font-bold text-gray-900">
              {folderId ? "Folder View" : "My Drive"}
            </h1>
          </div>

          <button
            onClick={() => setIsProfileOpen(true)}
            className="flex items-center gap-2 px-4 py-2 border rounded-lg hover:bg-gray-50 transition-colors"
          >
            <div className="w-8 h-8 rounded-full bg-blue-500 flex items-center justify-center text-white overflow-hidden border border-gray-100">
              {user?.picture ? (
                <img
                  src={user.picture}
                  alt={user.name}
                  className="w-full h-full object-cover"
                  // Optional: handle broken links
                  onError={(e) => {
                    e.currentTarget.style.display = "none";
                  }}
                />
              ) : (
                <span className="text-xs font-medium">
                  {user?.name?.[0]}
                  {user?.lastName?.[0]}
                </span>
              )}
            </div>
            <span className="font-medium text-gray-900">{user?.name}</span>
          </button>
        </header>

        {renderContent()}
      </main>

      {/* RIGHT DRAWER OVERLAY */}
      {isProfileOpen && (
        <div
          className="fixed inset-0 bg-black bg-opacity-25 z-40 transition-opacity"
          onClick={() => setIsProfileOpen(false)}
        />
      )}

      {/* RIGHT DRAWER PANEL */}
      <aside
        className={`
        fixed right-0 top-0 h-full w-80 bg-white shadow-2xl z-50 
        transform transition-transform duration-300 ease-in-out
        ${isProfileOpen ? "translate-x-0" : "translate-x-full"}
      `}
      >
        <div className="p-6">
          <div className="flex justify-between items-center mb-8">
            <h2 className="text-xl font-bold">Profile Details</h2>
            <button
              onClick={() => setIsProfileOpen(false)}
              className="text-gray-400 hover:text-gray-600"
            >
              ✕
            </button>
          </div>

          <div className="space-y-6">
            <div className="text-center">
              <div className="w-24 h-24 bg-gray-200 rounded-full mx-auto mb-4" />
              <h3 className="text-lg font-semibold">{user?.name}</h3>
              <p className="text-sm text-gray-500">{user?.email}</p>
            </div>

            <hr />
          </div>
        </div>
      </aside>

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
