import type { User } from "../context/AuthContext";
import { api } from "../lib/api";

export interface File {}
export interface Folder {
  ID: string;
  Name: string;
  OwnerID: string;
  Parent: Folder | null;
  ParentID: string | null;
  User: User | null;
  Folders: Folder[] | null;
  Files: File[] | null;
  IsDeleted: boolean;
  CreatedAt: string;
  UpdatedAt: string;
}

export const fetchFolders = async (
  parentId: string | null = null,
): Promise<Folder[]> => {
  // Use params to automatically format the URL: /folders?parentId=...
  const res = await api.get("/folders/", {
    params: parentId ? { parentId } : {},
  });

  return res.data;
};

export const createFolderApi = async (payload: {
  name: string;
  parentId?: string | null;
}) => {
  const res = await api.post("/folders/", payload);
  return res.data;
};
