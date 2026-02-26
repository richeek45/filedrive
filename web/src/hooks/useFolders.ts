import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  fetchFolders,
  createFolderApi,
  fetchFiles,
  downloadFileApi,
  moveToTrashApi,
  renameFileApi,
  syncPendingFileUploads,
} from "../services/folder.service";
import type { Folder } from "../services/folder.service";
import { uploadFileInParts } from "../lib/upload";
import { useState } from "react";

export const useFolders = (parentId: string | null = null) => {
  const [uploads, setUploads] = useState<Record<string, number>>({});
  const queryClient = useQueryClient();

  const foldersQuery = useQuery({
    // Important: Key must include parentId so TanStack treats each folder level as a unique cache
    queryKey: ["folders", parentId],
    queryFn: () => fetchFolders(parentId),
  });

  const syncQuery = useQuery({
    queryKey: ["files", "sync"],
    queryFn: async () => await syncPendingFileUploads(),
    staleTime: 60000,
    refetchOnWindowFocus: true,
  });

  const filesQuery = useQuery({
    queryKey: ["files", parentId],
    queryFn: () => fetchFiles(parentId),
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

  const moveToTrash = useMutation({
    mutationFn: (fileId: string) => moveToTrashApi(fileId),
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: ["files", parentId] }),
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
              ID: "temp-id",
              Name: newFolder.name,
              ParentID: newFolder.parentId ?? null,
              CreatedAt: new Date().toISOString(),
              UpdatedAt: new Date().toISOString(),
              OwnerID: "",
              Parent: null,
              User: null,
              IsDeleted: false,
              Folders: null,
              Files: null,
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
        console.log("percentage = ", percent);
        setUploads((prev) => ({ ...prev, [file.name]: percent }));
      }),
    onSuccess: (_, file) => {
      setUploads((prev) => {
        const newUploads = { ...prev };
        delete newUploads[file.name];
        return newUploads;
      });
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
    isLoading: foldersQuery.isLoading,
    createFolder: createFolderMutation.mutate,
    uploadFile: uploadFileMutation.mutateAsync, // mutateAsync is better for loops
    activeUploads: uploads,
    isSyncing: syncQuery.isPending,
    isUploading: uploadFileMutation.isPending,
    isCreating: createFolderMutation.isPending,
  };
};
