import { defineStore } from "pinia";
import { computed, ref } from "vue";
import pinia from "@/store";
import { getAccessToken, hasRefreshToken } from "@/utils/auth";
import { getBaseUrl } from "@/api/utils";
import { arcoMessage } from "@/globals";
import {
  type NoticeItem,
  getMyNoticeListAPI,
  getMyNoticeUnreadCountAPI,
  markMyNoticeReadAPI
} from "@/api/sysnotice";

let socket: WebSocket | null = null;
let reconnectTimer: number | null = null;

const categoryOrder = ["notice", "message", "backlog"] as const;

export const useNoticeStore = defineStore("notice", () => {
  const connected = ref(false);
  const loading = ref(false);
  const noticeList = ref<NoticeItem[]>([]);
  const unreadCount = ref(0);

  const noticeTabs = computed(() =>
    categoryOrder.map((category) => ({
      key: category,
      title: category,
      data: noticeList.value.filter((item) => item.category === category)
    }))
  );

  const bootstrap = async () => {
    if (!hasRefreshToken()) return;
    await Promise.allSettled([fetchMyNotices(), fetchUnreadCount()]);
    connect();
  };

  const fetchMyNotices = async () => {
    loading.value = true;
    try {
      const res = await getMyNoticeListAPI({ pageNum: 1, pageSize: 50 });
      noticeList.value = Array.isArray(res?.data?.list) ? res.data.list : [];
    } finally {
      loading.value = false;
    }
  };

  const fetchUnreadCount = async () => {
    const res = await getMyNoticeUnreadCountAPI();
    unreadCount.value = Number(res?.data?.unreadCount || 0);
  };

  const markRead = async (noticeIds: number[]) => {
    if (!noticeIds.length) return;
    await markMyNoticeReadAPI({ noticeIds });
    const idSet = new Set(noticeIds);
    noticeList.value = noticeList.value.map((item) =>
      idSet.has(item.id)
        ? {
            ...item,
            isRead: 1,
            readAt: new Date().toISOString()
          }
        : item
    );
    unreadCount.value = Math.max(0, noticeList.value.filter((item) => item.isRead !== 1).length);
  };

  const markAllRead = async () => {
    await markMyNoticeReadAPI({ all: true });
    noticeList.value = noticeList.value.map((item) => ({
      ...item,
      isRead: 1,
      readAt: item.readAt || new Date().toISOString()
    }));
    unreadCount.value = 0;
  };

  const connect = () => {
    if (socket && (socket.readyState === WebSocket.OPEN || socket.readyState === WebSocket.CONNECTING)) {
      return;
    }

    const tokenData = getAccessToken();
    if (!tokenData?.accessToken) return;

    const wsUrl = buildWsUrl(tokenData.accessToken);
    socket = new WebSocket(wsUrl);

    socket.onopen = () => {
      connected.value = true;
      clearReconnectTimer();
    };

    socket.onmessage = (event) => {
      try {
        const payload = JSON.parse(event.data);
        if (payload?.event === "notice:new" && payload?.data) {
          handleNewNotice(payload.data as NoticeItem);
        }
      } catch (error) {
        console.warn("parse notice ws message failed", error);
      }
    };

    socket.onclose = () => {
      connected.value = false;
      socket = null;
      scheduleReconnect();
    };

    socket.onerror = () => {
      connected.value = false;
    };
  };

  const disconnect = () => {
    clearReconnectTimer();
    connected.value = false;
    if (socket) {
      socket.close();
      socket = null;
    }
  };

  const handleNewNotice = (notice: NoticeItem) => {
    noticeList.value = [{ ...notice, isRead: 0 }, ...noticeList.value.filter((item) => item.id !== notice.id)].slice(0, 50);
    unreadCount.value += 1;
    arcoMessage("info", notice.title);
  };

  const buildWsUrl = (token: string) => {
    const rawBase = getBaseUrl() || window.location.origin;
    const wsBase = rawBase.replace(/^http/i, "ws").replace(/\/$/, "");
    return `${wsBase}/api/ws/notifications?token=${encodeURIComponent(token)}`;
  };

  const scheduleReconnect = () => {
    if (!hasRefreshToken()) return;
    if (reconnectTimer !== null) return;
    reconnectTimer = window.setTimeout(() => {
      reconnectTimer = null;
      connect();
    }, 3000);
  };

  const clearReconnectTimer = () => {
    if (reconnectTimer !== null) {
      window.clearTimeout(reconnectTimer);
      reconnectTimer = null;
    }
  };

  return {
    connected,
    loading,
    noticeList,
    unreadCount,
    noticeTabs,
    bootstrap,
    fetchMyNotices,
    fetchUnreadCount,
    markRead,
    markAllRead,
    connect,
    disconnect
  };
});

export function useNoticeStoreHook() {
  return useNoticeStore(pinia);
}
