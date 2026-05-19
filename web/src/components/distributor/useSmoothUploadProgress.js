/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/

import { useEffect, useRef, useState } from 'react';

/** 代理申请上传：平滑展示 axios 进度，避免 0→99 无过渡 */
export function useSmoothUploadProgress(uploadPct) {
    const [uploadDisplayPct, setUploadDisplayPct] = useState(null);
    const uploadTargetRef = useRef(null);
    const uploadDisplayRef = useRef(0);
    const uploadSmoothRafRef = useRef(null);

    uploadTargetRef.current = uploadPct;

    useEffect(() => {
        if (uploadPct == null) {
            if (uploadSmoothRafRef.current != null) {
                cancelAnimationFrame(uploadSmoothRafRef.current);
                uploadSmoothRafRef.current = null;
            }
            uploadDisplayRef.current = 0;
            setUploadDisplayPct(null);
            return;
        }

        let cancelled = false;
        const tick = () => {
            if (cancelled) return;
            const target = uploadTargetRef.current;
            if (target == null) return;
            const cur = uploadDisplayRef.current;
            if (cur < target - 0.01) {
                const next = Math.min(
                    target,
                    cur + Math.max(0.55, (target - cur) * 0.2),
                );
                uploadDisplayRef.current = next;
                setUploadDisplayPct(Math.round(next));
                uploadSmoothRafRef.current = requestAnimationFrame(tick);
            } else if (cur > target + 0.01) {
                uploadDisplayRef.current = target;
                setUploadDisplayPct(Math.round(target));
                uploadSmoothRafRef.current = requestAnimationFrame(tick);
            } else {
                uploadDisplayRef.current = target;
                setUploadDisplayPct(Math.round(target));
                uploadSmoothRafRef.current = null;
            }
        };

        uploadSmoothRafRef.current = requestAnimationFrame(tick);
        return () => {
            cancelled = true;
            if (uploadSmoothRafRef.current != null) {
                cancelAnimationFrame(uploadSmoothRafRef.current);
                uploadSmoothRafRef.current = null;
            }
        };
    }, [uploadPct]);

    return uploadDisplayPct;
}
