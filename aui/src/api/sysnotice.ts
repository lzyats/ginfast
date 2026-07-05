import { http } from "@/utils/http";
import { baseUrlApi } from "./utils";
import { BaseResult } from "./types";

export interface NoticeItem {
  id: number;
  title: string;
  content: string;
  category: "notice" | "message" | "backlog";
  sentAt: string | null;
  isRead?: number;
  readAt?: string | null;
  senderName?: string;
  recipientCount?: number;
  createdAt?: string;
  createdBy?: number;
  createdByName?: string;
}

export type NoticeListResult = BaseResult<{
  list: NoticeItem[];
  total: number;
}>;

export type NoticeUnreadCountResult = BaseResult<{
  unreadCount: number;
}>;

export const getSysNoticeListAPI = (params?: any) => {
  return http.request<NoticeListResult>("get", baseUrlApi("sysNotice/list"), { params });
};

export const getMyNoticeListAPI = (params?: any) => {
  return http.request<NoticeListResult>("get", baseUrlApi("sysNotice/my/list"), { params });
};

export const sendSysNoticeAPI = (data: { title: string; content: string; category: string; userIds?: number[] }) => {
  return http.request<BaseResult>("post", baseUrlApi("sysNotice/send"), { data });
};

export const markMyNoticeReadAPI = (data: { noticeIds?: number[]; all?: boolean }) => {
  return http.request<BaseResult>("put", baseUrlApi("sysNotice/my/read"), { data });
};

export const getMyNoticeUnreadCountAPI = () => {
  return http.request<NoticeUnreadCountResult>("get", baseUrlApi("sysNotice/my/unreadCount"));
};
