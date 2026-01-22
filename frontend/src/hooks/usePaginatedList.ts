import { useState, useCallback } from 'react';
import { message } from 'antd';
import type { PaginatedResponse } from '../types';

export interface UsePaginatedListOptions<T, F = Record<string, unknown>> {
  fetchApi: (params: { page: number; page_size: number } & F) => Promise<{ data: PaginatedResponse<T> }>;
  defaultPageSize?: number;
  onError?: (error: unknown) => void;
}

export interface UsePaginatedListReturn<T, F = Record<string, unknown>> {
  loading: boolean;
  data: T[];
  total: number;
  page: number;
  pageSize: number;
  setPage: (page: number) => void;
  setPageSize: (size: number) => void;
  fetchData: (filters?: F) => Promise<void>;
  refresh: () => Promise<void>;
  handlePageChange: (page: number, pageSize: number) => void;
}

export function usePaginatedList<T, F = Record<string, unknown>>(
  options: UsePaginatedListOptions<T, F>
): UsePaginatedListReturn<T, F> {
  const { fetchApi, defaultPageSize = 10, onError } = options;

  const [loading, setLoading] = useState(false);
  const [data, setData] = useState<T[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(defaultPageSize);
  const [lastFilters, setLastFilters] = useState<F | undefined>();

  const fetchData = useCallback(async (filters?: F) => {
    setLoading(true);
    try {
      const params = { page, page_size: pageSize, ...filters } as { page: number; page_size: number } & F;
      const res = await fetchApi(params);
      setData(res.data.items);
      setTotal(res.data.total);
      setLastFilters(filters);
    } catch (error) {
      if (onError) {
        onError(error);
      } else {
        console.error('Failed to fetch data:', error);
        message.error('Failed to load data');
      }
    } finally {
      setLoading(false);
    }
  }, [fetchApi, page, pageSize, onError]);

  const refresh = useCallback(async () => {
    await fetchData(lastFilters);
  }, [fetchData, lastFilters]);

  const handlePageChange = useCallback((newPage: number, newPageSize: number) => {
    setPage(newPage);
    setPageSize(newPageSize);
  }, []);

  return {
    loading,
    data,
    total,
    page,
    pageSize,
    setPage,
    setPageSize,
    fetchData,
    refresh,
    handlePageChange,
  };
}
