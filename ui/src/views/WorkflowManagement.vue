<template>
  <div class="min-h-screen bg-gray-50 dark:bg-gray-900">
    <div class="max-w-7xl mx-auto px-6 py-8">
      <div class="mb-8 flex items-center justify-between">
        <div>
          <router-link to="/" class="text-blue-600 dark:text-blue-400 hover:underline mb-4 inline-block"> ← 返回首页 </router-link>
          <h1 class="text-3xl font-bold text-gray-900 dark:text-white">工作流管理</h1>
          <p class="text-gray-600 dark:text-gray-400 mt-2">管理和可视化 Agent 工作流（演示模式）</p>
        </div>
        <div class="flex items-center gap-3">
          <span v-if="DEMO_MODE" class="px-3 py-1 bg-amber-100 dark:bg-amber-900/30 text-amber-700 dark:text-amber-300 text-sm rounded-lg"> 🎭 演示模式 </span>
          <button v-if="!DEMO_MODE" @click="showCreateDialog = true" class="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg font-medium transition-colors flex items-center gap-2">
            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
            </svg>
            创建工作流
          </button>
        </div>
      </div>

      <WorkflowList :workflows="workflows" :loading="loading" @execute="handleExecute" @edit="handleEdit" @delete="handleDelete" />

      <!-- 创建/编辑对话框 -->
      <div v-if="showCreateDialog || showEditDialog" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50" @click.self="closeDialogs">
        <div class="bg-white dark:bg-gray-800 rounded-xl shadow-xl max-w-2xl w-full mx-4 max-h-[90vh] overflow-y-auto">
          <div class="p-6 border-b border-gray-200 dark:border-gray-700">
            <h2 class="text-2xl font-bold text-gray-900 dark:text-white">
              {{ showEditDialog ? "编辑工作流" : "创建工作流" }}
            </h2>
          </div>

          <div class="p-6 space-y-4">
            <div>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2"> 工作流名称 </label>
              <input
                v-model="formData.name"
                type="text"
                placeholder="输入工作流名称"
                class="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              />
            </div>

            <div>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2"> 描述 </label>
              <textarea
                v-model="formData.description"
                rows="3"
                placeholder="输入工作流描述"
                class="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              ></textarea>
            </div>

            <div>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2"> 工作流步骤 </label>
              <div class="space-y-2">
                <div v-for="(step, index) in formData.steps" :key="index" class="flex items-center gap-2">
                  <span class="text-gray-500 dark:text-gray-400 w-6">{{ index + 1 }}.</span>
                  <input v-model="step.name" type="text" placeholder="步骤名称" class="flex-1 px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white" />
                  <button @click="removeStep(index)" class="p-2 text-red-600 hover:bg-red-50 dark:hover:bg-red-900/20 rounded-lg">
                    <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
                    </svg>
                  </button>
                </div>
                <button @click="addStep" class="w-full px-4 py-2 border-2 border-dashed border-gray-300 dark:border-gray-600 rounded-lg text-gray-600 dark:text-gray-400 hover:border-blue-500 hover:text-blue-600 dark:hover:text-blue-400 transition-colors">
                  + 添加步骤
                </button>
              </div>
            </div>
          </div>

          <div class="p-6 border-t border-gray-200 dark:border-gray-700 flex justify-end gap-3">
            <button @click="closeDialogs" class="px-4 py-2 border border-gray-300 dark:border-gray-600 text-gray-700 dark:text-gray-300 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors">取消</button>
            <button @click="showEditDialog ? updateWorkflow() : createWorkflow()" class="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg transition-colors">
              {{ showEditDialog ? "保存" : "创建" }}
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, reactive } from "vue";
import WorkflowList from "../components/Workflow/WorkflowList.vue";
import { useAsterClient } from "../composables/useAsterClient";
import { DEMO_MODE, demoWorkflows } from "../config/demoData";

const { client } = useAsterClient();
const workflows = ref<any[]>([]);

console.log("🎭 演示模式:", DEMO_MODE ? "启用（使用本地数据）" : "禁用（连接后端API）");

const loading = ref(false);
const showCreateDialog = ref(false);
const showEditDialog = ref(false);
const editingWorkflowId = ref<string | null>(null);

const formData = reactive({
  name: "",
  description: "",
  steps: [{ id: "s1", name: "", status: "pending" }],
});

// Create - 创建工作流
const createWorkflow = async () => {
  if (!formData.name.trim()) {
    alert("请输入工作流名称");
    return;
  }

  if (DEMO_MODE) {
    alert("演示模式下不支持创建新工作流\n\n您可以查看和执行现有的演示工作流\n\n提示：设置 VITE_DEMO_MODE=false 启用完整功能");
    closeDialogs();
    return;
  }

  try {
    loading.value = true;
    const response = await client.workflows.create({
      name: formData.name,
      description: formData.description,
      version: "1.0.0",
      steps: formData.steps
        .filter((s) => s.name.trim())
        .map((s, i) => ({
          id: `s${i + 1}`,
          name: s.name,
          type: "task",
          config: {},
        })),
    });

    if (response.success) {
      await loadWorkflows();
      closeDialogs();
      alert(`工作流 "${formData.name}" 创建成功！`);
    } else {
      alert(`创建失败: ${response.message || "未知错误"}`);
    }
  } catch (error: any) {
    console.error("创建工作流失败:", error);
    alert(`创建失败: ${error.message}`);
  } finally {
    loading.value = false;
  }
};

// Read - 加载工作流列表
const loadWorkflows = async () => {
  try {
    loading.value = true;

    if (DEMO_MODE) {
      // 演示模式：使用 UI 本地数据
      workflows.value = JSON.parse(JSON.stringify(demoWorkflows));
    } else {
      // 生产模式：从后端 API 获取
      const response = await client.workflows.list();
      if (response.success && response.data) {
        workflows.value = response.data.map((w: any) => ({
          ...w,
          steps: w.steps || [],
          status: w.status || "idle",
        }));
      }
    }
  } catch (error: any) {
    console.error("加载工作流失败:", error);
    // 失败时使用演示数据作为后备
    workflows.value = JSON.parse(JSON.stringify(demoWorkflows));
  } finally {
    loading.value = false;
  }
};

// Update - 更新工作流
const updateWorkflow = async () => {
  if (!formData.name.trim()) {
    alert("请输入工作流名称");
    return;
  }

  if (DEMO_MODE) {
    alert("演示模式下不支持编辑工作流\n\n提示：设置 VITE_DEMO_MODE=false 启用完整功能");
    closeDialogs();
    return;
  }

  try {
    loading.value = true;
    const response = await client.workflows.update(editingWorkflowId.value!, {
      name: formData.name,
      description: formData.description,
      steps: formData.steps
        .filter((s) => s.name.trim())
        .map((s, i) => ({
          id: `s${i + 1}`,
          name: s.name,
          type: "task",
          config: {},
        })),
    });

    if (response.success) {
      await loadWorkflows();
      closeDialogs();
      alert(`工作流 "${formData.name}" 更新成功！`);
    } else {
      alert(`更新失败: ${response.message || "未知错误"}`);
    }
  } catch (error: any) {
    console.error("更新工作流失败:", error);
    alert(`更新失败: ${error.message}`);
  } finally {
    loading.value = false;
  }
};

// Delete - 删除工作流
const handleDelete = async (workflow: any) => {
  if (DEMO_MODE) {
    alert("演示模式下不支持删除工作流\n\n提示：设置 VITE_DEMO_MODE=false 启用完整功能");
    return;
  }

  if (confirm(`确定要删除工作流 "${workflow.name}" 吗？`)) {
    try {
      loading.value = true;
      const response = await client.workflows.delete(workflow.id);
      if (response.success) {
        await loadWorkflows();
        alert("工作流已删除");
      } else {
        alert(`删除失败: ${response.message || "未知错误"}`);
      }
    } catch (error: any) {
      console.error("删除工作流失败:", error);
      alert(`删除失败: ${error.message}`);
    } finally {
      loading.value = false;
    }
  }
};

// Execute - 执行工作流（演示模式：模拟执行）
const handleExecute = async (workflow: any) => {
  const index = workflows.value.findIndex((w) => w.id === workflow.id);
  if (index !== -1) {
    workflows.value[index].status = "running";
    alert(`工作流 "${workflow.name}" 开始执行！\n\n演示模式：将模拟执行过程`);

    // 模拟执行过程
    let currentStep = 0;
    const interval = setInterval(() => {
      if (currentStep < workflows.value[index].steps.length) {
        workflows.value[index].steps[currentStep].status = "completed";
        currentStep++;
        if (currentStep < workflows.value[index].steps.length) {
          workflows.value[index].steps[currentStep].status = "running";
        }
      } else {
        clearInterval(interval);
        workflows.value[index].status = "completed";
        setTimeout(() => {
          alert(`工作流 "${workflow.name}" 执行完成！`);
        }, 500);
      }
    }, 1500);
  }
};

// Edit - 编辑工作流
const handleEdit = (workflow: any) => {
  if (DEMO_MODE) {
    alert("演示模式下不支持编辑工作流\n\n提示：设置 VITE_DEMO_MODE=false 启用完整功能");
    return;
  }

  editingWorkflowId.value = workflow.id;
  formData.name = workflow.name;
  formData.description = workflow.description;
  formData.steps = workflow.steps.map((s: any) => ({ ...s }));
  showEditDialog.value = true;
};

// 步骤管理
const addStep = () => {
  formData.steps.push({
    id: `s${formData.steps.length + 1}`,
    name: "",
    status: "pending",
  });
};

const removeStep = (index: number) => {
  if (formData.steps.length > 1) {
    formData.steps.splice(index, 1);
  } else {
    alert("至少需要保留一个步骤");
  }
};

const closeDialogs = () => {
  showCreateDialog.value = false;
  showEditDialog.value = false;
  editingWorkflowId.value = null;
  formData.name = "";
  formData.description = "";
  formData.steps = [{ id: "s1", name: "", status: "pending" }];
};

onMounted(() => {
  loadWorkflows();
});
</script>
