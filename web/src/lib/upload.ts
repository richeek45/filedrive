import { api } from "./api";

const CHUNK_SIZE = 5 * 1024 * 1024; // 5MB
const CONCURRENCY_LIMIT = 4;
const MAX_RETRIES = 3;

export const uploadFileInParts = async (
  file: File,
  parentId: string | null,
) => {
  try {
    // 1. Initiate
    console.log("intiating from the uploadFileInParts function");
    const totalParts = Math.ceil(file.size / CHUNK_SIZE);

    const { data: initData } = await api.post("/files/uploads/initiate", {
      fileName: file.name,
      contentType: file.type,
      parentId: parentId,
      size: file.size,
      totalChunks: totalParts,
    });

    const { uploadId, key } = initData;

    // Create a queue of part numbers
    // const queue = Array.from({ length: totalParts }, (_, i) => i + 1);
    const completedParts: { PartNumber: number; ETag: string }[] = [];

    const finishedNumbers = new Set(
      completedParts.map((p: any) => p.PartNumber),
    );
    const queue = Array.from({ length: totalParts }, (_, i) => i + 1).filter(
      (num) => !finishedNumbers.has(num),
    );

    const allCompletedParts = [...completedParts];

    // Helper: Upload a single part with retry logic
    const uploadPartWithRetry = async (
      partNumber: number,
      attempt = 1,
    ): Promise<void> => {
      try {
        const start = (partNumber - 1) * CHUNK_SIZE;
        const end = Math.min(start + CHUNK_SIZE, file.size);
        const blob = file.slice(start, end);

        // A. Get Presigned URL from Go
        const { data: urlData } = await api.post(
          "/files/uploads/presign-part",
          {
            uploadId,
            key,
            partNumber,
          },
        );

        // B. PUT to S3
        const uploadResponse = await fetch(urlData.url, {
          method: "PUT",
          body: blob,
        });

        if (!uploadResponse.ok) throw new Error("S3 Upload Failed");

        const etag =
          uploadResponse.headers.get("ETag")?.replace(/"/g, "") || "";
        completedParts.push({ PartNumber: partNumber, ETag: etag });
      } catch (err) {
        if (attempt < MAX_RETRIES) {
          console.warn(
            `Part ${partNumber} failed. Retrying (${attempt}/${MAX_RETRIES})...`,
          );
          return uploadPartWithRetry(partNumber, attempt + 1);
        }
        throw err;
      }
    };

    // 2. Concurrency Worker Pool
    const worker = async () => {
      while (queue.length > 0) {
        const partNumber = queue.shift();
        if (partNumber !== undefined) {
          await uploadPartWithRetry(partNumber);
        }
      }
    };

    // Fire off workers based on CONCURRENCY_LIMIT
    await Promise.all(Array.from({ length: CONCURRENCY_LIMIT }, worker));

    // 3. Finalize
    completedParts.sort((a, b) => a.PartNumber - b.PartNumber);
    await api.post("/files/uploads/complete", {
      uploadId,
      key,
      parts: completedParts,
      parentId,
      // add parts
    });

    return true;
  } catch (error) {
    console.error(`Fatal error uploading ${file.name}:`, error);
    return false;
  }
};
