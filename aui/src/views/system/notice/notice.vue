<template>
  <div class="notice-page">
    <a-card class="send-card" title="发送通知" :bordered="false">
      <a-form :model="form" layout="vertical">
        <a-row :gutter="16">
          <a-col :xs="24" :md="8">
            <a-form-item field="category" label="通知类型">
              <a-select v-model="form.category" allow-clear>
                <a-option value="notice">通知</a-option>
                <a-option value="message">消息</a-option>
                <a-option value="backlog">待办</a-option>
              </a-select>
            </a-form-item>
          </a-col>
          <a-col :xs="24" :md="16">
            <a-form-item field="title" label="通知标题">
              <a-input v-model="form.title" placeholder="请输入通知标题" allow-clear />
            </a-form-item>
          </a-col>
        </a-row>

        <a-form-item field="content" label="通知内容">
          <a-textarea
            v-model="form.content"
            placeholder="请输入通知内容"
            :auto-size="{ minRows: 4, maxRows: 8 }"
            allow-clear
          />
        </a-form-item>

        <a-form-item field="sendType" label="发送范围">
          <a-radio-group v-model="form.sendType" type="button">
            <a-radio value="all">所有用户</a-radio>
            <a-radio value="selected">指定用户</a-radio>
          </a-radio-group>
        </a-form-item>

        <div v-if="form.sendType === 'selected'" class="receiver-panel">
          <a-card title="选择接收用户" :bordered="false">
            <a-space wrap class="receiver-search">
              <a-input
                v-model="userQuery.name"
                placeholder="用户名或昵称"
                allow-clear
                style="width: 220px"
                @keyup.enter="getUsers"
              />
              <a-input
                v-model="userQuery.phone"
                placeholder="手机号"
                allow-clear
                style="width: 180px"
                @keyup.enter="getUsers"
              />
              <a-select v-model="userQuery.status" placeholder="状态" allow-clear style="width: 120px">
                <a-option :value="1">启用</a-option>
                <a-option :value="0">禁用</a-option>
              </a-select>
              <a-button type="primary" @click="getUsers">查询</a-button>
              <a-button @click="resetUserQuery">重置</a-button>
            </a-space>

            <div class="receiver-summary">
              查询结果 {{ userQuery.total }} 人，
              已选用户 {{ selectedUserIds.length }} 人
            </div>

            <a-table
              row-key="id"
              :data="userTableData"
              :loading="userLoading"
              :pagination="userPagination"
              :row-selection="rowSelection"
              :scroll="{ x: 920, y: 320 }"
              v-model:selected-keys="selectedUserIds"
              @page-change="handleUserPageChange"
              @page-size-change="handleUserPageSizeChange"
            >
              <template #columns>
                <a-table-column title="ID" data-index="id" :width="80" />
                <a-table-column title="用户名" data-index="userName" :width="180" ellipsis tooltip />
                <a-table-column title="昵称" data-index="nickName" :width="160" ellipsis tooltip />
                <a-table-column title="手机号" data-index="phone" :width="160" />
                <a-table-column title="状态" :width="100">
                  <template #cell="{ record }">
                    <a-tag :color="record.status === 1 ? 'green' : 'red'">
                      {{ record.status === 1 ? "启用" : "禁用" }}
                    </a-tag>
                  </template>
                </a-table-column>
                <a-table-column title="部门" data-index="departmentName" :width="180" ellipsis tooltip />
              </template>
            </a-table>
          </a-card>
        </div>

        <a-space>
          <a-button type="primary" :loading="sending" @click="handleSend">
            {{ form.sendType === "all" ? "发送给所有用户" : "发送给选中用户" }}
          </a-button>
          <a-button @click="handleReset">清空</a-button>
        </a-space>
      </a-form>
    </a-card>

    <a-card class="list-card" title="发送记录" :bordered="false">
      <div class="toolbar">
        <a-space wrap>
          <a-input v-model="query.title" placeholder="按标题搜索" allow-clear style="width: 220px" />
          <a-select v-model="query.category" allow-clear placeholder="全部类型" style="width: 140px">
            <a-option value="notice">通知</a-option>
            <a-option value="message">消息</a-option>
            <a-option value="backlog">待办</a-option>
          </a-select>
          <a-button type="primary" @click="getList">查询</a-button>
          <a-button @click="resetQuery">重置</a-button>
        </a-space>
      </div>

      <a-table :data="tableData" :loading="loading" :pagination="pagination" row-key="id" @page-change="handlePageChange">
        <template #columns>
          <a-table-column title="ID" data-index="id" :width="80" />
          <a-table-column title="类型" data-index="category" :width="110">
            <template #cell="{ record }">
              <a-tag :color="tagColorMap[record.category]">{{ categoryLabelMap[record.category] || record.category }}</a-tag>
            </template>
          </a-table-column>
          <a-table-column title="标题" data-index="title" :width="220" ellipsis tooltip />
          <a-table-column title="内容" data-index="content" ellipsis tooltip />
          <a-table-column title="接收人数" data-index="recipientCount" :width="110" />
          <a-table-column title="发送人" data-index="createdByName" :width="140" ellipsis tooltip />
          <a-table-column title="发送时间" data-index="sentAt" :width="180">
            <template #cell="{ record }">
              {{ formatTime(record.sentAt || record.createdAt) }}
            </template>
          </a-table-column>
        </template>
      </a-table>
    </a-card>
  </div>
</template>

<script setup lang="ts">
import { arcoMessage, formatTime } from "@/globals";
import { getSysNoticeListAPI, getSysNoticeUsersAPI, sendSysNoticeAPI, type NoticeSelectableUserItem } from "@/api/sysnotice";
import { useNoticeStoreHook } from "@/store/modules/notice";

const noticeStore = useNoticeStoreHook();
const loading = ref(false);
const sending = ref(false);
const userLoading = ref(false);
const tableData = ref<any[]>([]);
const userTableData = ref<NoticeSelectableUserItem[]>([]);
const selectedUserIds = ref<number[]>([]);

const rowSelection = reactive({
  type: "checkbox",
  showCheckedAll: true,
  onlyCurrent: false
});

const query = reactive({
  title: "",
  category: "",
  pageNum: 1,
  pageSize: 10,
  total: 0
});

const userQuery = reactive({
  name: "",
  phone: "",
  status: null as number | null,
  pageNum: 1,
  pageSize: 10,
  total: 0
});

const form = reactive({
  title: "",
  content: "",
  category: "notice",
  sendType: "all" as "all" | "selected"
});

const categoryLabelMap: Record<string, string> = {
  notice: "通知",
  message: "消息",
  backlog: "待办"
};

const tagColorMap: Record<string, string> = {
  notice: "arcoblue",
  message: "green",
  backlog: "orange"
};

const pagination = computed(() => ({
  current: query.pageNum,
  pageSize: query.pageSize,
  total: query.total,
  showTotal: true
}));

const userPagination = computed(() => ({
  current: userQuery.pageNum,
  pageSize: userQuery.pageSize,
  total: userQuery.total,
  showPageSize: true,
  showTotal: true,
  pageSizeOptions: ["10", "20", "50", "100"]
}));

const getList = async () => {
  loading.value = true;
  try {
    const res = await getSysNoticeListAPI({
      title: query.title,
      category: query.category,
      pageNum: query.pageNum,
      pageSize: query.pageSize
    });
    tableData.value = res?.data?.list || [];
    query.total = Number(res?.data?.total || 0);
  } catch {
    arcoMessage("error", "获取通知记录失败");
  } finally {
    loading.value = false;
  }
};

const getUsers = async () => {
  userLoading.value = true;
  try {
    const res = await getSysNoticeUsersAPI({
      name: userQuery.name,
      phone: userQuery.phone,
      status: userQuery.status,
      pageNum: userQuery.pageNum,
      pageSize: userQuery.pageSize,
      order: "id desc"
    });
    userTableData.value = Array.isArray(res?.data?.list) ? res.data.list : [];
    userQuery.total = Number(res?.data?.total || 0);
  } catch {
    arcoMessage("error", "获取用户列表失败");
  } finally {
    userLoading.value = false;
  }
};

const handleSend = async () => {
  if (!form.title.trim()) {
    arcoMessage("warning", "请输入通知标题");
    return;
  }
  if (!form.content.trim()) {
    arcoMessage("warning", "请输入通知内容");
    return;
  }
  if (form.sendType === "selected" && selectedUserIds.value.length === 0) {
    arcoMessage("warning", "请先选择接收用户");
    return;
  }

  sending.value = true;
  try {
    await sendSysNoticeAPI({
      title: form.title,
      content: form.content,
      category: form.category,
      userIds: form.sendType === "selected" ? selectedUserIds.value : undefined
    });
    arcoMessage("success", "通知已发送");
    handleReset();
    await Promise.all([getList(), noticeStore.fetchMyNotices(), noticeStore.fetchUnreadCount()]);
  } catch {
    arcoMessage("error", "发送通知失败");
  } finally {
    sending.value = false;
  }
};

const handleReset = () => {
  form.title = "";
  form.content = "";
  form.category = "notice";
  form.sendType = "all";
  selectedUserIds.value = [];
  resetUserQuery(false);
};

const resetQuery = () => {
  query.title = "";
  query.category = "";
  query.pageNum = 1;
  getList();
};

const resetUserQuery = (shouldLoad = true) => {
  userQuery.name = "";
  userQuery.phone = "";
  userQuery.status = null;
  userQuery.pageNum = 1;
  userQuery.pageSize = 10;
  if (shouldLoad) {
    getUsers();
  }
};

const handlePageChange = (page: number) => {
  query.pageNum = page;
  getList();
};

const handleUserPageChange = (page: number) => {
  userQuery.pageNum = page;
  getUsers();
};

const handleUserPageSizeChange = (pageSize: number) => {
  userQuery.pageSize = pageSize;
  userQuery.pageNum = 1;
  getUsers();
};

watch(
  () => form.sendType,
  value => {
    if (value === "selected" && userTableData.value.length === 0) {
      getUsers();
    }
  }
);

onMounted(() => {
  getList();
});
</script>

<style lang="scss" scoped>
.notice-page {
  box-sizing: border-box;
  height: 100%;
  overflow-y: auto;
  padding: $padding;
}

.send-card,
.list-card {
  margin-bottom: 16px;
}

.toolbar,
.receiver-search {
  margin-bottom: 16px;
}

.receiver-panel {
  margin-bottom: 16px;
}

.receiver-summary {
  margin-bottom: 12px;
  color: var(--color-text-2);
}

:deep(.receiver-panel .arco-card-body) {
  overflow: hidden;
}
</style>
