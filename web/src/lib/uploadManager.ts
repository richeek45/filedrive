import { get, set } from "idb-keyval";

export const useUploadManager = () => {
  const registerFileHandle = async (
    fileId: string,
    handle: FileSystemFileHandle,
    type: string,
  ) => {
    await set(`handle_${fileId}`, { handle, type });
  };

  const getPersistentFile = async (fileId: string): Promise<File | null> => {
    try {
      const data = await get<{ handle: FileSystemFileHandle; type: string }>(
        `handle_${fileId}`,
      );

      if (!data) return null;

      const { handle, type } = data;
      const options = { mode: "read" as const };
      if ((await handle.queryPermission(options)) !== "granted") {
        if ((await handle.requestPermission(options)) !== "granted") {
          return null;
        }
      }

      const file = await handle.getFile();
      if (!file.type && type) {
        return new File([file], file.name, { type });
      }
      return file;
    } catch (err) {
      console.error("Failed to restore file handle", err);
      return null;
    }
  };

  return { registerFileHandle, getPersistentFile };
};
