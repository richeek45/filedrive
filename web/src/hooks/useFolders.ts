import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  fetchFolders,
  createFolderApi,
  fetchFiles,
  downloadFileApi,
  moveToTrashApi,
  renameFileApi,
  syncPendingFileUploads,
  renameFolderApi,
  shareFilesApi,
} from "../services/folder.service";
import type { Folder } from "../services/folder.service";
import { uploadFileInParts } from "../lib/upload";
import { useState } from "react";

export const useFolders = (
  parentId: string | null = null,
  isTrash: boolean = false,
) => {
  const [uploads, setUploads] = useState<Record<string, number>>({});
  const queryClient = useQueryClient();

  const foldersQuery = useQuery({
    // Important: Key must include parentId so TanStack treats each folder level as a unique cache
    queryKey: ["folders", parentId, isTrash],
    queryFn: () => fetchFolders(parentId, isTrash),
  });

  const syncQuery = useQuery({
    queryKey: ["files", "sync"],
    queryFn: async () => await syncPendingFileUploads(),
    staleTime: 60000,
    refetchOnWindowFocus: true,
  });

  const filesQuery = useQuery({
    queryKey: ["files", parentId, isTrash],
    queryFn: () => fetchFiles(parentId, isTrash),
    enabled: syncQuery.isSuccess || syncQuery.isError,
  });

  const downloadFile = useMutation({
    mutationFn: async (fileId: string) => {
      const data = await downloadFileApi(fileId);
      window.open(data.url, "_blank");
    },
    onSuccess: () => console.log("downloaded file successfully"),
  });

  const renameFile = useMutation({
    mutationFn: ({ id, name }: { id: string; name: string }) =>
      renameFileApi(id, name),
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: ["files", parentId] }),
  });

  const renameFolder = useMutation({
    mutationFn: ({ id, name }: { id: string; name: string }) =>
      renameFolderApi(id, name),
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: ["folders", parentId] }),
  });

  const moveToTrash = useMutation({
    mutationFn: (fileId: string) => moveToTrashApi(fileId),
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: ["files", parentId, isTrash] }),
  });

  const createFolderMutation = useMutation({
    mutationFn: createFolderApi,

    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["folders", parentId] });
    },

    // Optional: Optimistic update
    onMutate: async (newFolder) => {
      const queryKey = ["folders", parentId];
      await queryClient.cancelQueries({ queryKey });

      const previousFolders = queryClient.getQueryData<Folder[]>(queryKey);

      if (previousFolders) {
        queryClient.setQueryData<Folder[]>(
          ["folders"],
          [
            ...previousFolders,
            {
              id: "temp-id",
              name: newFolder.name,
              parentId: newFolder.parentId ?? null,
              createdAt: new Date().toISOString(),
              updatedAt: new Date().toISOString(),
              folders: null,
              files: null,
            },
          ],
        );
      }

      return { previousFolders };
    },

    onError: (_, __, context) => {
      if (context?.previousFolders) {
        queryClient.setQueryData(
          ["folders", parentId],
          context.previousFolders,
        );
      }
    },
  });

  const uploadFileMutation = useMutation({
    mutationFn: (file: File) =>
      uploadFileInParts(file, parentId, (percent) => {
        setUploads((prev) => ({ ...prev, [file.name]: percent }));
      }),
    onSuccess: (_, file) => {
      setUploads((prev) => {
        const newUploads = { ...prev };
        delete newUploads[file.name];
        return newUploads;
      });
      queryClient.invalidateQueries({ queryKey: ["files", parentId, isTrash] });
      queryClient.invalidateQueries({
        queryKey: ["folders", parentId, isTrash],
      });
      queryClient.invalidateQueries({ queryKey: ["files", "sync"] });
      queryClient.invalidateQueries({ queryKey: ["authUser"] });
    },
  });

  const shareFileMutation = useMutation({
    mutationFn: ({
      fileId,
      folderId,
      emails,
      permission,
    }: {
      fileId: string;
      folderId: string;
      emails: string[];
      permission: string;
    }) => shareFilesApi({ fileId, folderId, emails, permission }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["files", parentId] });
      queryClient.invalidateQueries({ queryKey: ["folders", parentId] });
      queryClient.invalidateQueries({ queryKey: ["files", "sync"] });
    },
  });

  return {
    folders: foldersQuery.data ?? [],
    files: filesQuery.data ?? [],
    moveToTrash: moveToTrash.mutate,
    downloadFile: downloadFile.mutate,
    renameFile: renameFile.mutate,
    renameFolder: renameFolder.mutate,
    isLoading: foldersQuery.isLoading,
    createFolder: createFolderMutation.mutate,
    uploadFile: uploadFileMutation.mutateAsync, // mutateAsync is better for loops
    shareFile: shareFileMutation.mutate,
    activeUploads: uploads,
    isSyncing: syncQuery.isPending,
    isUploading: uploadFileMutation.isPending,
    isCreating: createFolderMutation.isPending,
  };
};
