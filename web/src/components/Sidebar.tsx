// components/Sidebar.tsx
import React, { useState, useRef } from 'react';
import { 
  Menu, 
  Home, 
  Computer, 
  Clock, 
  Star, 
  Trash2,
  Plus,
  FolderPlus,
  Upload,
  FolderUp
} from 'lucide-react';

interface SidebarProps {
  onNewFolder: () => void;
  onFileUpload: (files: FileList) => void;
  onFolderUpload: (files: FileList) => void;
}

const Sidebar: React.FC<SidebarProps> = ({ 
  onNewFolder, 
  onFileUpload, 
  onFolderUpload 
}) => {
  const [isNewMenuOpen, setIsNewMenuOpen] = useState(false);
  const [isCollapsed, setIsCollapsed] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const folderInputRef = useRef<HTMLInputElement>(null);

  const menuItems = [
    { icon: Home, label: 'Home' },
    { icon: Computer, label: 'My Drive' },
    { icon: Clock, label: 'Recent' },
    { icon: Star, label: 'Starred' },
    { icon: Trash2, label: 'Trash' },
  ];

  const handleNewClick = () => {
    setIsNewMenuOpen(!isNewMenuOpen);
  };

  const handleFileUpload = (event: React.ChangeEvent<HTMLInputElement>) => {
    const files = event.target.files;
    if (files && files.length > 0) {
      onFileUpload(files);
    }
    setIsNewMenuOpen(false);
  };

  const handleFolderUpload = (event: React.ChangeEvent<HTMLInputElement>) => {
    const files = event.target.files;
    if (files && files.length > 0) {
      onFolderUpload(files);
    }
    setIsNewMenuOpen(false);
  };

  const handleNewFolder = () => {
    onNewFolder();
    setIsNewMenuOpen(false);
  };

  return (
    <div 
      className={`relative h-screen bg-white border-r border-gray-200 transition-all duration-300 ${
        isCollapsed ? 'w-20' : 'w-64'
      }`}
    >
      {/* Toggle button */}
      <button
        onClick={() => setIsCollapsed(!isCollapsed)}
        className="absolute -right-3 top-6 bg-white border border-gray-200 rounded-full p-1 hover:bg-gray-50"
      >
        <Menu size={16} />
      </button>

      <div className="p-4">
        {/* New button */}
        <div className="relative mb-4">
          <button
            onClick={handleNewClick}
            className={`flex items-center justify-center gap-2 w-full bg-blue-600 hover:bg-blue-700 text-white font-medium py-2 px-4 rounded-lg transition-colors ${
              isCollapsed ? 'px-2' : ''
            }`}
          >
            <Plus size={20} />
            {!isCollapsed && <span>New</span>}
          </button>

          {/* New menu dropdown */}
          {isNewMenuOpen && (
            <>
              <div 
                className="fixed inset-0 z-10"
                onClick={() => setIsNewMenuOpen(false)}
              />
              <div className={`absolute top-full left-0 mt-1 w-48 bg-white rounded-lg shadow-lg border border-gray-200 py-1 z-20 ${
                isCollapsed ? 'left-12' : ''
              }`}>
                <button
                  onClick={handleNewFolder}
                  className="flex items-center gap-3 w-full px-4 py-2 text-left hover:bg-gray-50 transition-colors"
                >
                  <FolderPlus size={18} className="text-gray-600" />
                  <span>New folder</span>
                </button>
                
                <button
                  onClick={() => fileInputRef.current?.click()}
                  className="flex items-center gap-3 w-full px-4 py-2 text-left hover:bg-gray-50 transition-colors"
                >
                  <Upload size={18} className="text-gray-600" />
                  <span>File upload</span>
                </button>
                <input
                  ref={fileInputRef}
                  type="file"
                  multiple
                  className="hidden"
                  onChange={handleFileUpload}
                />
                
                <button
                  onClick={() => folderInputRef.current?.click()}
                  className="flex items-center gap-3 w-full px-4 py-2 text-left hover:bg-gray-50 transition-colors"
                >
                  <FolderUp size={18} className="text-gray-600" />
                  <span>Folder upload</span>
                </button>
                <input
                  ref={folderInputRef}
                  type="file"
                //   webkitdirectory=""
                //   directory=""
                  multiple
                  className="hidden"
                  onChange={handleFolderUpload}
                />
              </div>
            </>
          )}
        </div>

        {/* Navigation menu */}
        <nav className="space-y-1">
          {menuItems.map((item) => (
            <button
              key={item.label}
              className={`flex items-center gap-3 w-full px-3 py-2 text-gray-700 rounded-lg hover:bg-gray-100 transition-colors ${
                isCollapsed ? 'justify-center' : ''
              }`}
            >
              <item.icon size={20} />
              {!isCollapsed && <span>{item.label}</span>}
            </button>
          ))}
        </nav>
      </div>
    </div>
  );
};

export default Sidebar;