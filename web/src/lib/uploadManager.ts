import { get, set, del } from "idb-keyval";

export const useUploadManager = () => {
  const registerFileHandle = async (
    fileId: string,
    handle: FileSystemFileHandle,
  ) => {
    await set(`handle_${fileId}`, handle);
  };

  const getPersistentFile = async (fileId: string): Promise<File | null> => {
    try {
      const handle = await get<FileSystemFileHandle>(`handle_${fileId}`);
      if (!handle) return null;

      // Verify permissions (Browser usually requires a user gesture for this)
      const options = { mode: "read" as const };
      if ((await handle.queryPermission(options)) !== "granted") {
        if ((await handle.requestPermission(options)) !== "granted") {
          return null;
        }
      }

      return await handle.getFile();
    } catch (err) {
      console.error("Failed to restore file handle", err);
      return null;
    }
  };

  return { registerFileHandle, getPersistentFile };
};
