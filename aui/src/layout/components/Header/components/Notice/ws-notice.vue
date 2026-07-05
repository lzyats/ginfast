<template>
  <div class="notice-panel">
    <div class="notice-actions">
      <a-button type="text" size="mini" @click="markAllRead">全部已读</a-button>
    </div>
    <a-tabs :default-active-key="current" :active-key="current" @tab-click="onTab">
      <a-tab-pane v-for="item in noticeTabs" :key="item.key">
        <template #title>{{ `${tabTitleMap[item.key]}(${item.data.length})` }}</template>
        <div v-for="content in item.data" :key="content.id" class="notice" :class="{ unread: content.isRead !== 1 }" @click="handleRead(content)">
          <a-image width="36" height="36" fit="cover" :src="myImage" class="notice_img" />
          <div class="content margin-left-text">
            <div class="title-row">
              <span class="nickname">{{ content.title }}</span>
              <span class="time margin-left-text">{{ formatTime(content.sentAt || content.readAt) }}</span>
            </div>
            <div class="sender">{{ content.senderName || "系统" }}</div>
            <div class="context">{{ content.content }}</div>
          </div>
        </div>
        <a-empty v-if="item.data.length === 0" />
      </a-tab-pane>
    </a-tabs>
  </div>
</template>

<script setup lang="ts">
import myImage from "@/assets/img/my-image.jpg";
import { formatTime } from "@/globals";
import { useNoticeStoreHook } from "@/store/modules/notice";
import { storeToRefs } from "pinia";

const noticeStore = useNoticeStoreHook();
const { noticeTabs } = storeToRefs(noticeStore);

const tabTitleMap: Record<string, string> = {
  notice: "通知",
  message: "消息",
  backlog: "待办"
};

const current = ref<string>("notice");

const onTab = (key: string) => {
  current.value = key;
};

const handleRead = async (item: any) => {
  if (item.isRead === 1) return;
  await noticeStore.markRead([item.id]);
};

const markAllRead = async () => {
  await noticeStore.markAllRead();
};
</script>

<style lang="scss" scoped>
.notice-panel {
  min-width: 320px;
}

.notice-actions {
  display: flex;
  justify-content: flex-end;
  margin-bottom: 8px;
}

.notice {
  display: flex;
  align-items: flex-start;
  margin-bottom: $margin;
  padding: 10px;
  border-radius: 8px;
  cursor: pointer;

  &:hover {
    background: $color-fill-1;
  }

  &.unread {
    background: rgba(var(--primary-6), 0.08);
  }

  .notice_img {
    width: 36px;
    height: 36px;
    border-radius: 50%;
  }

  .content {
    width: 240px;

    .title-row {
      display: flex;
      align-items: center;
      justify-content: space-between;
    }

    .nickname {
      font-size: $font-size-body-3;
      color: $color-text-1;
      font-weight: 600;
    }

    .sender,
    .time {
      font-size: $font-size-body-1;
      color: $color-text-3;
    }

    .context {
      margin-top: 4px;
      font-size: $font-size-body-3;
      color: $color-text-2;
      line-height: 1.5;
      word-break: break-all;
    }
  }
}

.margin-left-text {
  margin-left: $margin-text;
}

:deep(.arco-tabs-content .arco-tabs-content-list) {
  display: unset;
}
</style>
