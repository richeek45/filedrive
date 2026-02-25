export {};

declare global {
  interface Window {
    showOpenFilePicker(options?: {
      multiple?: boolean;
      excludeAcceptAllOption?: boolean;
      types?: Array<{
        description?: string;
        accept: Record<string, string[]>;
      }>;
    }): Promise<FileSystemFileHandle[]>;
  }

  interface FileSystemFileHandle extends FileSystemHandle {
    kind: "file";
    getFile(): Promise<File>;
    createWritable(
      options?: FileSystemCreateWritableOptions,
    ): Promise<FileSystemWritableFileStream>;
    queryPermission(
      descriptor?: FileSystemHandlePermissionDescriptor,
    ): Promise<PermissionState>;
    requestPermission(
      descriptor?: FileSystemHandlePermissionDescriptor,
    ): Promise<PermissionState>;
  }

  interface FileSystemHandle {
    kind: "file" | "directory";
    name: string;
    isSameEntry(other: FileSystemHandle): Promise<boolean>;
  }
}
