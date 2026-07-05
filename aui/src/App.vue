<template>
    <div>
        <router-view />
    </div>
</template>

<script setup lang="ts">
import { watch } from "vue";
import { useThemeMethods } from "@/hooks/useThemeMethods";
import { useNoticeStoreHook } from "@/store/modules/notice";
import { useSysConfigStore } from "@/store/modules/sys-config";
import { hasRefreshToken } from "@/utils/auth";

const onTheme = () => {
    const { initTheme } = useThemeMethods();
    initTheme();
};
onTheme();

const setDefaultFavicon = () => {
    const defaultIconUrl = "src/assets/sys/default.ico";
    const links = document.querySelectorAll("link[rel='icon']");
    links.forEach(link => {
        link.remove();
    });

    const link = document.createElement("link");
    link.rel = "icon";
    link.href = defaultIconUrl;
    document.head.appendChild(link);
};

const setFavicon = (iconUrl: string) => {
    if (!iconUrl) {
        setDefaultFavicon();
        return;
    }

    const testImg = new Image();
    testImg.onload = () => {
        const links = document.querySelectorAll("link[rel='icon']");
        links.forEach(link => {
            link.remove();
        });

        const link = document.createElement("link");
        link.rel = "icon";
        link.href = iconUrl;
        document.head.appendChild(link);
    };
    testImg.onerror = () => {
        console.warn(`图标加载失败: ${iconUrl}，将使用默认图标`);
        setDefaultFavicon();
    };
    testImg.src = iconUrl;
};

const setTitle = (title: string) => {
    if (title) {
        document.title = title;
    }
};

const initSysConfig = () => {
    const sysConfigStore = useSysConfigStore();
    sysConfigStore
        .getConfig()
        .then(() => {
            setFavicon(sysConfigStore.systemIcon);
            setTitle(sysConfigStore.systemConfig.systemName);
        })
        .catch((error: unknown) => {
            console.warn("获取系统配置失败，将使用默认配置:", error);
        });
};

const sysConfigStore = useSysConfigStore();
watch(
    () => sysConfigStore.systemIcon,
    newIcon => {
        if (newIcon) {
            setFavicon(newIcon);
        }
    },
    { immediate: true }
);

watch(
    () => sysConfigStore.systemConfig.systemName,
    newName => {
        if (newName) {
            setTitle(newName);
        }
    },
    { immediate: true }
);

initSysConfig();

if (hasRefreshToken()) {
    useNoticeStoreHook().bootstrap().catch((error: unknown) => {
        console.warn("初始化通知失败:", error);
    });
}
</script>

<style lang="scss" scoped></style>
