import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { fetchFolders, createFolderApi } from "../services/folder.service";
import type { Folder } from "../services/folder.service";

export const useFolders = (parentId: string | null = null) => {
  const queryClient = useQueryClient();

  const foldersQuery = useQuery({
    // Important: Key must include parentId so TanStack treats each folder level as a unique cache
    queryKey: ["folders", parentId],
    queryFn: () => fetchFolders(parentId),
  });

  const createFolderMutation = useMutation({
    mutationFn: createFolderApi,

    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["folders", parentId] });
    },

    // Optional: Optimistic update
    onMutate: async (newFolder) => {
      await queryClient.cancelQueries({ queryKey: ["folders"] });

      const previousFolders = queryClient.getQueryData<Folder[]>(["folders"]);

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
        queryClient.setQueryData(["folders"], context.previousFolders);
      }
    },
  });

  return {
    folders: foldersQuery.data ?? [],
    isLoading: foldersQuery.isLoading,
    createFolder: createFolderMutation.mutate,
    isCreating: createFolderMutation.isPending,
  };
};
