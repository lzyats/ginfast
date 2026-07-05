<template>
  <div class="snow-page">
    <div class="snow-inner">
      <s-layout-tools>
        <template #left>
          <a-space wrap>
            <a-input v-model="form.name" placeholder="请输入参数名称" allow-clear />
            <a-input v-model="form.code" placeholder="请输入参数标识" allow-clear />
            <a-select v-model="form.paramType" placeholder="参数类型" style="width: 140px" allow-clear>
              <a-option v-for="item in paramTypeOptions" :key="item.value" :value="item.value">
                {{ item.label }}
              </a-option>
            </a-select>
            <a-select v-model="form.status" placeholder="启用状态" style="width: 120px" allow-clear>
              <a-option v-for="item in openState" :key="item.value" :value="item.value">
                {{ item.name }}
              </a-option>
            </a-select>
            <a-button type="primary" @click="search">
              <template #icon><icon-search /></template>
              <span>查询</span>
            </a-button>
            <a-button @click="reset">
              <template #icon><icon-refresh /></template>
              <span>重置</span>
            </a-button>
          </a-space>
        </template>
        <template #right>
          <a-space wrap>
            <a-button type="primary" @click="onAdd" v-hasPerm="['system:param:add']">
              <template #icon><icon-plus /></template>
              <span>新增</span>
            </a-button>
          </a-space>
        </template>
      </s-layout-tools>

      <a-table
        row-key="id"
        :data="paramList"
        :bordered="{ cell: true }"
        :loading="loading"
        :scroll="{ x: '100%', y: '100%', minWidth: 1180 }"
        :pagination="pagination"
      >
        <template #columns>
          <a-table-column title="ID" data-index="id" :width="70" align="center" />
          <a-table-column title="参数名称" data-index="name" :width="150" />
          <a-table-column title="参数标识" data-index="code" :width="200" />
          <a-table-column title="参数类型" :width="120" align="center">
            <template #cell="{ record }">
              <a-tag size="small" color="arcoblue">{{ getParamTypeLabel(record.paramType) }}</a-tag>
            </template>
          </a-table-column>
          <a-table-column title="参数值" :width="280" :ellipsis="true" :tooltip="true">
            <template #cell="{ record }">
              {{ renderParamValue(record) }}
            </template>
          </a-table-column>
          <a-table-column title="状态" :width="100" align="center">
            <template #cell="{ record }">
              <a-tag bordered size="small" color="arcoblue" v-if="record.status === 1">启用</a-tag>
              <a-tag bordered size="small" color="red" v-else>禁用</a-tag>
            </template>
          </a-table-column>
          <a-table-column title="描述" data-index="description" :ellipsis="true" :width="180" :tooltip="true" />
          <a-table-column title="创建时间" data-index="createdAt" :width="180">
            <template #cell="{ record }">
              {{ record.createdAt ? formatTime(record.createdAt) : "" }}
            </template>
          </a-table-column>
          <a-table-column title="操作" :width="200" align="center" :fixed="isMobile ? '' : 'right'">
            <template #cell="{ record }">
              <a-space>
                <a-link @click="onUpdate(record)" v-hasPerm="['system:param:edit']">
                  <template #icon><icon-edit /></template>
                  <span>修改</span>
                </a-link>
                <a-popconfirm type="warning" content="确定删除该参数吗?" @ok="onDelete(record)">
                  <a-link status="danger" v-hasPerm="['system:param:delete']">
                    <template #icon><icon-delete /></template>
                    <span>删除</span>
                  </a-link>
                </a-popconfirm>
              </a-space>
            </template>
          </a-table-column>
        </template>
      </a-table>
    </div>

    <a-modal
      :width="layoutMode.width"
      v-model:visible="open"
      @close="afterClose"
      :on-before-ok="handleOk"
      @cancel="afterClose"
    >
      <template #title>{{ title }}</template>
      <a-form ref="formRef" :layout="layoutMode.layout" auto-label-width :rules="rules" :model="addForm">
        <a-form-item field="name" label="参数名称" validate-trigger="blur">
          <a-input v-model="addForm.name" placeholder="请输入参数名称" allow-clear />
        </a-form-item>
        <a-form-item field="code" label="参数标识" validate-trigger="blur">
          <a-input v-model="addForm.code" placeholder="请输入参数唯一标识" allow-clear />
        </a-form-item>
        <a-form-item field="paramType" label="参数类型" validate-trigger="change">
          <a-select v-model="addForm.paramType" placeholder="请选择参数类型" @change="onParamTypeChange">
            <a-option v-for="item in paramTypeOptions" :key="item.value" :value="item.value">
              {{ item.label }}
            </a-option>
          </a-select>
        </a-form-item>

        <a-form-item v-if="addForm.paramType === 'text'" field="value" label="参数值" validate-trigger="blur">
          <a-textarea
            v-model="addForm.value"
            placeholder="请输入文本参数值"
            :auto-size="{ minRows: 3, maxRows: 6 }"
            allow-clear
          />
        </a-form-item>

        <a-form-item v-if="addForm.paramType === 'number'" field="value" label="参数值" validate-trigger="blur">
          <a-input-number v-model="numericValue" placeholder="请输入数值参数值" style="width: 100%" :precision="6" />
        </a-form-item>

        <template v-if="addForm.paramType === 'select'">
          <a-form-item field="options" label="下拉选项" validate-trigger="change">
            <div class="option-editor">
              <div class="option-editor__header">
                <span>可分别设置显示名和实际值，实际值留空时默认等于显示名，实际值不能重复。</span>
                <a-button size="small" type="outline" @click="addOptionRow">
                  <template #icon><icon-plus /></template>
                  <span>新增选项</span>
                </a-button>
              </div>
              <div v-for="(row, index) in optionRows" :key="row.key" class="option-editor__row">
                <a-input v-model="row.label" placeholder="显示名" allow-clear @change="syncSelectValue" />
                <a-input
                  v-model="row.value"
                  placeholder="实际值，不填则默认同显示名"
                  allow-clear
                  @change="syncSelectValue"
                />
                <div class="option-editor__actions">
                  <a-button type="text" size="small" :disabled="index === 0" @click="moveOptionRow(index, 'up')">
                    上移
                  </a-button>
                  <a-button
                    type="text"
                    size="small"
                    :disabled="index === optionRows.length - 1"
                    @click="moveOptionRow(index, 'down')"
                  >
                    下移
                  </a-button>
                </div>
                <a-button
                  status="danger"
                  type="text"
                  :disabled="optionRows.length === 1"
                  @click="removeOptionRow(index)"
                >
                  <template #icon><icon-delete /></template>
                </a-button>
              </div>
            </div>
          </a-form-item>

          <a-form-item field="value" label="默认值" validate-trigger="change">
            <a-select v-model="addForm.value" placeholder="请选择默认值" allow-clear>
              <a-option v-for="item in selectOptions" :key="item.value" :value="item.value">
                {{ item.label }}
              </a-option>
            </a-select>
          </a-form-item>
        </template>

        <a-form-item v-if="addForm.paramType === 'upload'" field="value" label="上传文件">
          <FileUpload v-model="addForm.value" :max-count="1" title="上传参数文件" />
        </a-form-item>

        <a-form-item field="description" label="描述" validate-trigger="blur">
          <a-textarea v-model="addForm.description" placeholder="请输入描述" allow-clear />
        </a-form-item>
        <a-form-item field="status" label="状态" validate-trigger="change">
          <a-switch type="round" :checked-value="1" :unchecked-value="0" v-model="addForm.status">
            <template #checked>启用</template>
            <template #unchecked>禁用</template>
          </a-switch>
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { deepClone } from "@/utils";
import { formatTime } from "@/globals";
import FileUpload from "@/components/upload/file-upload.vue";
import {
  addParamAPI,
  deleteParamAPI,
  getParamListAPI,
  type ParamAddParams,
  type ParamListParams,
  type ParamType,
  type ParamUpdateParams,
  type SystemParam,
  updateParamAPI
} from "@/api/sysparam";
import { useDevicesSize } from "@/hooks/useDevicesSize";

interface SelectOptionItem {
  label: string;
  value: string;
}

interface SelectOptionRow {
  key: string;
  label: string;
  value: string;
}

const { isMobile } = useDevicesSize();
const layoutMode = computed(() => {
  const info = {
    mobile: {
      width: "95%",
      layout: "vertical"
    },
    desktop: {
      width: "48%",
      layout: "horizontal"
    }
  };
  return isMobile.value ? info.mobile : info.desktop;
});

const paramTypeOptions: Array<{ label: string; value: ParamType }> = [
  { label: "文本", value: "text" },
  { label: "数值", value: "number" },
  { label: "下拉选择", value: "select" },
  { label: "上传", value: "upload" }
];

const openState = ref(dictFilter("status"));
const form = ref<ParamListParams>({
  name: "",
  code: "",
  paramType: undefined,
  status: undefined
});

const search = () => {
  currentPage.value = 1;
  getParamList();
};

const reset = () => {
  form.value = {
    name: "",
    code: "",
    paramType: undefined,
    status: undefined
  };
  currentPage.value = 1;
  getParamList();
};

const open = ref(false);
const title = ref("");
const rules = {
  name: [{ required: true, message: "请输入参数名称" }],
  code: [{ required: true, message: "请输入参数唯一标识" }],
  paramType: [{ required: true, message: "请选择参数类型" }],
  status: [{ required: true, message: "请选择状态" }]
};

const defaultAddForm = (): ParamAddParams & { id?: number } => ({
  name: "",
  code: "",
  value: "",
  paramType: "text",
  options: "",
  status: 1,
  description: ""
});

const createOptionRow = (label = "", value = ""): SelectOptionRow => ({
  key: `${Date.now()}-${Math.random().toString(16).slice(2)}`,
  label,
  value
});

const addForm = ref<ParamAddParams & { id?: number }>(defaultAddForm());
const optionRows = ref<SelectOptionRow[]>([createOptionRow()]);
const formRef = ref();

const numericValue = computed<number | undefined>({
  get() {
    if (!addForm.value.value) return undefined;
    const value = Number(addForm.value.value);
    return Number.isNaN(value) ? undefined : value;
  },
  set(value) {
    addForm.value.value = value === undefined || value === null ? "" : String(value);
  }
});

const selectOptions = computed<SelectOptionItem[]>(() =>
  optionRows.value
    .map(row => {
      const label = row.label.trim();
      const value = (row.value.trim() || label).trim();
      return { label, value };
    })
    .filter(item => item.label && item.value)
);

const duplicateOptionValues = computed<string[]>(() => {
  const seen = new Set<string>();
  const duplicates = new Set<string>();

  selectOptions.value.forEach(item => {
    if (seen.has(item.value)) {
      duplicates.add(item.value);
      return;
    }
    seen.add(item.value);
  });

  return Array.from(duplicates);
});

watch(selectOptions, () => {
  syncSelectValue();
});

const onAdd = () => {
  open.value = true;
  title.value = "新增参数";
};

const addOptionRow = () => {
  optionRows.value.push(createOptionRow());
};

const removeOptionRow = (index: number) => {
  if (optionRows.value.length === 1) return;
  optionRows.value.splice(index, 1);
  syncSelectValue();
};

const moveOptionRow = (index: number, direction: "up" | "down") => {
  const targetIndex = direction === "up" ? index - 1 : index + 1;
  if (targetIndex < 0 || targetIndex >= optionRows.value.length) return;

  const rows = [...optionRows.value];
  const [current] = rows.splice(index, 1);
  rows.splice(targetIndex, 0, current);
  optionRows.value = rows;
};

const resetOptionRows = () => {
  optionRows.value = [createOptionRow()];
};

const setOptionRows = (options: SelectOptionItem[]) => {
  optionRows.value = options.length > 0
    ? options.map(item => createOptionRow(item.label, item.value))
    : [createOptionRow()];
};

const syncSelectValue = () => {
  if (addForm.value.paramType !== "select") return;
  if (!addForm.value.value) return;
  const exists = selectOptions.value.some(item => item.value === addForm.value.value);
  if (!exists) {
    addForm.value.value = "";
  }
};

const onParamTypeChange = (value: ParamType) => {
  addForm.value.paramType = value;
  if (value !== "select") {
    resetOptionRows();
    addForm.value.options = "";
  }
  if (value === "number") {
    const numeric = Number(addForm.value.value);
    addForm.value.value = Number.isNaN(numeric) ? "" : String(numeric);
    return;
  }
  if (value === "upload") {
    const trimmed = addForm.value.value?.trim?.() ?? "";
    if (!(trimmed.startsWith("[") && trimmed.endsWith("]"))) {
      addForm.value.value = "";
    }
    return;
  }
  if (value === "select") {
    if (optionRows.value.length === 0) {
      resetOptionRows();
    }
    addForm.value.value = "";
  }
};

const handleOk = async () => {
  const state = await formRef.value.validate();
  if (state) return false;

  if (addForm.value.paramType === "select" && selectOptions.value.length === 0) {
    arcoMessage("error", "请先配置下拉选项");
    return false;
  }

  if (duplicateOptionValues.value.length > 0) {
    arcoMessage("error", `下拉选项实际值不能重复：${duplicateOptionValues.value.join("、")}`);
    return false;
  }

  try {
    const baseData = {
      name: addForm.value.name,
      code: addForm.value.code,
      value: addForm.value.value,
      paramType: addForm.value.paramType,
      options: addForm.value.paramType === "select" ? JSON.stringify(selectOptions.value) : "",
      status: addForm.value.status,
      description: addForm.value.description
    };

    if (addForm.value.id) {
      const updateData: ParamUpdateParams = {
        id: addForm.value.id,
        ...baseData
      };
      await updateParamAPI(updateData);
      arcoMessage("success", "修改参数成功");
    } else {
      const addData: ParamAddParams = baseData;
      await addParamAPI(addData);
      arcoMessage("success", "新增参数成功");
    }
    getParamList();
    return true;
  } catch (error) {
    console.error("操作失败:", error);
    return false;
  }
};

const afterClose = () => {
  formRef.value?.resetFields();
  addForm.value = defaultAddForm();
  resetOptionRows();
};

const onUpdate = (record: SystemParam) => {
  title.value = "修改参数";
  addForm.value = {
    ...deepClone(record),
    paramType: (record.paramType || "text") as ParamType,
    options: record.options || ""
  };
  setOptionRows(parseStoredOptions(record.options || ""));
  open.value = true;
};

const onDelete = async (record: SystemParam) => {
  try {
    await deleteParamAPI({ id: record.id });
    arcoMessage("success", "删除成功");
    getParamList();
  } catch (error) {
    console.error("删除失败:", error);
    arcoMessage("error", "删除失败");
  }
};

const loading = ref(false);
const currentPage = ref(1);
const pageSize = ref(10);
const total = ref(0);
const pagination = ref({
  current: currentPage,
  pageSize: pageSize,
  total: total,
  showPageSize: true,
  showTotal: true,
  onChange: (page: number) => {
    currentPage.value = page;
    getParamList();
  },
  onPageSizeChange: (size: number) => {
    pageSize.value = size;
    currentPage.value = 1;
    getParamList();
  }
});

const paramList = ref<SystemParam[]>([]);
const getParamList = async () => {
  loading.value = true;
  try {
    const params: ParamListParams = {
      pageNum: currentPage.value,
      pageSize: pageSize.value,
      order: "id desc",
      ...form.value
    };
    const res = await getParamListAPI(params);
    if (res.data) {
      paramList.value = res.data.list || [];
      total.value = res.data.total || 0;
    }
  } catch (error) {
    console.error("获取参数列表失败:", error);
    arcoMessage("error", "获取参数列表失败");
  } finally {
    loading.value = false;
  }
};

const getParamTypeLabel = (type?: string) => {
  return paramTypeOptions.find(item => item.value === type)?.label || "文本";
};

const renderParamValue = (record: SystemParam) => {
  if (record.paramType === "upload") {
    try {
      const files = JSON.parse(record.value || "[]");
      if (Array.isArray(files) && files.length > 0) {
        return files.map(item => item.name || item.url || "已上传文件").join("，");
      }
    } catch (error) {
      return record.value;
    }
    return "";
  }

  if (record.paramType === "select") {
    const options = parseStoredOptions(record.options || "");
    const matched = options.find(item => item.value === record.value);
    return matched ? `${matched.label} (${matched.value})` : record.value;
  }

  return record.value;
};

const parseStoredOptions = (options: string): SelectOptionItem[] => {
  if (!options) return [];
  try {
    const parsed = JSON.parse(options);
    if (Array.isArray(parsed)) {
      return parsed
        .map(item => ({
          label: String(item?.label ?? "").trim(),
          value: String(item?.value ?? "").trim()
        }))
        .filter(item => item.label && item.value);
    }
  } catch (error) {
    return options
      .split("\n")
      .map(line => line.trim())
      .filter(Boolean)
      .map(line => {
        const [labelPart, valuePart] = line.split("|");
        const label = labelPart?.trim?.() || "";
        const value = valuePart?.trim?.() || label;
        return { label, value };
      })
      .filter(item => item.label && item.value);
  }
  return [];
};

getParamList();
</script>

<style lang="scss" scoped>
.option-editor {
  display: flex;
  width: 100%;
  flex-direction: column;
  gap: 12px;
}

.option-editor__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  color: var(--color-text-3);
  font-size: 12px;
}

.option-editor__row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr) 108px 44px;
  gap: 8px;
  align-items: center;
}

.option-editor__actions {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 4px;
}

@media (max-width: 768px) {
  .option-editor__header {
    align-items: flex-start;
    flex-direction: column;
  }

  .option-editor__row {
    grid-template-columns: 1fr;
  }
}
</style>
