import type { User } from "../context/AuthContext";
import { api } from "../lib/api";

export interface File {
  ID: string;
  Name: string;
  OwnerID: string;
  FolderID: string;
  Size: number;
  MimeType: string;
  // S3 metadata
  BucketName: string;
  ObjectKey: string;
  S3UploadID: string;
  ETag: string;
  // Upload tracking
  UploadStatus: string;
  TotalChunks: number;
  UploadedChunks: number;
  UploadedPartNumbers: number;

  IsDeleted: boolean;
  // Folder *Folder `gorm:"foreignKey:FolderID;references:ID;constraint:OnDelete:CASCADE;" json:"-"`

  CreatedAt: number;
  UpdatedAt: number;
  DeletedAt: number;
}

export interface Folder {
  id: string;
  name: string;
  parentId: string | null;
  folders: Folder[] | null;
  files: File[] | null;
  createdAt: string;
  updatedAt: string;
}

export const fetchFolders = async (
  parentId: string | null = null,
  isTrash: boolean,
): Promise<Folder[]> => {
  // Use params to automatically format the URL: /folders?parentId=...
  const res = await api.get("/folders/", {
    params: parentId ? { parentId, isTrash } : { isTrash },
  });

  return res.data;
};

export const fetchFiles = async (
  parentId: string | null = null,
  isTrash: boolean,
): Promise<File[]> => {
  const res = await api.get("/files/", {
    params: parentId ? { parentId, isTrash } : { isTrash },
  });

  return res.data;
};

export const syncPendingFileUploads = async () => {
  const { data } = await api.get("files/sync-active-uploads");
  return data;
};

export const downloadFileApi = async (fileId: string) => {
  const res = await api.get(`/files/${fileId}/download`);
  return res.data;
};

export const renameFileApi = async (fileId: string, name: string) => {
  const res = await api.patch(`/files/${fileId}/rename`, { name });
  return res.data;
};

export const renameFolderApi = async (folderId: string, name: string) => {
  const res = await api.patch(`/folders/${folderId}/rename`, { name });
  return res.data;
};

export const moveToTrashApi = (fileId: string) =>
  api.patch(`/files/${fileId}/trash`);

export const createFolderApi = async (payload: {
  name: string;
  parentId?: string | null;
}) => {
  const res = await api.post("/folders/", payload);
  return res.data;
};
