import { useState } from 'react';
import { Save, Plus, AlertCircle } from 'lucide-react';
import { Header } from '../components/Header';
import { usePricing, useUpdatePricing } from '../hooks/useApi';
import type { ModelPricing } from '../api/types';

export function Settings() {
  const { data: pricing, isLoading, refetch } = usePricing();
  const updatePricing = useUpdatePricing();

  const [editingModel, setEditingModel] = useState<string | null>(null);
  const [editValues, setEditValues] = useState<Partial<ModelPricing>>({});
  const [newModel, setNewModel] = useState({
    model: '',
    input_price_per_m: 0,
    output_price_per_m: 0,
    currency: 'USD',
  });
  const [showAddForm, setShowAddForm] = useState(false);

  const handleEdit = (model: string, current: ModelPricing) => {
    setEditingModel(model);
    setEditValues(current);
  };

  const handleSave = async (model: string) => {
    if (!editValues.input_price_per_m || !editValues.output_price_per_m) return;

    try {
      await updatePricing.mutateAsync({
        model,
        input_price_per_m: editValues.input_price_per_m,
        output_price_per_m: editValues.output_price_per_m,
        currency: editValues.currency || 'USD',
      });
      setEditingModel(null);
      refetch();
    } catch (error) {
      console.error('Failed to update pricing:', error);
    }
  };

  const handleAddModel = async () => {
    if (!newModel.model) return;

    try {
      await updatePricing.mutateAsync(newModel);
      setShowAddForm(false);
      setNewModel({
        model: '',
        input_price_per_m: 0,
        output_price_per_m: 0,
        currency: 'USD',
      });
      refetch();
    } catch (error) {
      console.error('Failed to add model:', error);
    }
  };

  return (
    <div className="flex flex-col h-full">
      <Header title="Settings" subtitle="系统配置" />

      <div className="flex-1 overflow-auto p-6">
        <div className="max-w-4xl mx-auto space-y-6">
          {/* Pricing Configuration */}
          <div className="card">
            <div className="flex items-center justify-between mb-4">
              <div>
                <h3 className="text-lg font-semibold text-[var(--color-text-primary)]">
                  模型价格配置
                </h3>
                <p className="text-sm text-[var(--color-text-muted)]">
                  配置各模型的输入/输出 Token 价格（每百万 Token）
                </p>
              </div>
              <button
                onClick={() => setShowAddForm(true)}
                className="flex items-center gap-2 px-3 py-1.5 rounded-lg text-sm bg-[var(--color-primary)] text-white hover:bg-[var(--color-primary-dark)] transition-colors"
              >
                <Plus className="w-4 h-4" />
                添加模型
              </button>
            </div>

            {isLoading ? (
              <div className="space-y-2">
                {[...Array(3)].map((_, i) => (
                  <div
                    key={i}
                    className="h-16 bg-[var(--color-background)] rounded-lg animate-pulse-slow"
                  />
                ))}
              </div>
            ) : pricing && Object.keys(pricing).length > 0 ? (
              <div className="space-y-2">
                {/* Add Form */}
                {showAddForm && (
                  <div className="p-4 bg-[var(--color-background)] rounded-lg border border-[var(--color-primary)]">
                    <div className="grid grid-cols-4 gap-4">
                      <input
                        type="text"
                        placeholder="模型名称"
                        value={newModel.model}
                        onChange={(e) =>
                          setNewModel({ ...newModel, model: e.target.value })
                        }
                        className="px-3 py-2 bg-[var(--color-surface)] border border-[var(--color-border)] rounded-lg text-sm text-[var(--color-text-primary)] placeholder:text-[var(--color-text-muted)] focus:outline-none focus:border-[var(--color-primary)]"
                      />
                      <input
                        type="number"
                        placeholder="输入价格"
                        value={newModel.input_price_per_m || ''}
                        onChange={(e) =>
                          setNewModel({
                            ...newModel,
                            input_price_per_m: parseFloat(e.target.value) || 0,
                          })
                        }
                        className="px-3 py-2 bg-[var(--color-surface)] border border-[var(--color-border)] rounded-lg text-sm text-[var(--color-text-primary)] placeholder:text-[var(--color-text-muted)] focus:outline-none focus:border-[var(--color-primary)]"
                      />
                      <input
                        type="number"
                        placeholder="输出价格"
                        value={newModel.output_price_per_m || ''}
                        onChange={(e) =>
                          setNewModel({
                            ...newModel,
                            output_price_per_m: parseFloat(e.target.value) || 0,
                          })
                        }
                        className="px-3 py-2 bg-[var(--color-surface)] border border-[var(--color-border)] rounded-lg text-sm text-[var(--color-text-primary)] placeholder:text-[var(--color-text-muted)] focus:outline-none focus:border-[var(--color-primary)]"
                      />
                      <div className="flex items-center gap-2">
                        <button
                          onClick={handleAddModel}
                          className="flex-1 flex items-center justify-center gap-2 px-3 py-2 rounded-lg text-sm bg-[var(--color-success)] text-white hover:opacity-90 transition-opacity"
                        >
                          <Save className="w-4 h-4" />
                          保存
                        </button>
                        <button
                          onClick={() => setShowAddForm(false)}
                          className="px-3 py-2 rounded-lg text-sm bg-[var(--color-surface-elevated)] text-[var(--color-text-secondary)] hover:bg-[var(--color-border)] transition-colors"
                        >
                          取消
                        </button>
                      </div>
                    </div>
                  </div>
                )}

                {/* Model List */}
                {Object.entries(pricing).map(([model, modelPricing]) => (
                  <div
                    key={model}
                    className="p-4 bg-[var(--color-background)] rounded-lg"
                  >
                    {editingModel === model ? (
                      <div className="grid grid-cols-4 gap-4 items-center">
                        <span className="font-mono text-sm text-[var(--color-text-primary)]">
                          {model}
                        </span>
                        <input
                          type="number"
                          value={editValues.input_price_per_m || ''}
                          onChange={(e) =>
                            setEditValues({
                              ...editValues,
                              input_price_per_m: parseFloat(e.target.value) || 0,
                            })
                          }
                          className="px-3 py-2 bg-[var(--color-surface)] border border-[var(--color-border)] rounded-lg text-sm text-[var(--color-text-primary)] focus:outline-none focus:border-[var(--color-primary)]"
                        />
                        <input
                          type="number"
                          value={editValues.output_price_per_m || ''}
                          onChange={(e) =>
                            setEditValues({
                              ...editValues,
                              output_price_per_m: parseFloat(e.target.value) || 0,
                            })
                          }
                          className="px-3 py-2 bg-[var(--color-surface)] border border-[var(--color-border)] rounded-lg text-sm text-[var(--color-text-primary)] focus:outline-none focus:border-[var(--color-primary)]"
                        />
                        <div className="flex items-center gap-2">
                          <button
                            onClick={() => handleSave(model)}
                            className="flex-1 flex items-center justify-center gap-2 px-3 py-2 rounded-lg text-sm bg-[var(--color-success)] text-white hover:opacity-90 transition-opacity"
                          >
                            <Save className="w-4 h-4" />
                            保存
                          </button>
                          <button
                            onClick={() => setEditingModel(null)}
                            className="px-3 py-2 rounded-lg text-sm bg-[var(--color-surface-elevated)] text-[var(--color-text-secondary)] hover:bg-[var(--color-border)] transition-colors"
                          >
                            取消
                          </button>
                        </div>
                      </div>
                    ) : (
                      <div className="grid grid-cols-4 gap-4 items-center">
                        <span className="font-mono text-sm text-[var(--color-text-primary)]">
                          {model}
                        </span>
                        <span className="text-sm text-[var(--color-text-secondary)]">
                          输入: ${modelPricing.input_price_per_m}/M
                        </span>
                        <span className="text-sm text-[var(--color-text-secondary)]">
                          输出: ${modelPricing.output_price_per_m}/M
                        </span>
                        <div className="flex items-center justify-end gap-2">
                          <button
                            onClick={() => handleEdit(model, modelPricing)}
                            className="px-3 py-1.5 rounded-lg text-sm bg-[var(--color-surface-elevated)] text-[var(--color-text-secondary)] hover:bg-[var(--color-border)] transition-colors"
                          >
                            编辑
                          </button>
                        </div>
                      </div>
                    )}
                  </div>
                ))}
              </div>
            ) : (
              <div className="flex flex-col items-center justify-center py-8 text-[var(--color-text-muted)]">
                <AlertCircle className="w-12 h-12 mb-4" />
                <p>暂无价格配置</p>
              </div>
            )}
          </div>

          {/* API Configuration */}
          <div className="card">
            <h3 className="text-lg font-semibold text-[var(--color-text-primary)] mb-2">
              API 配置
            </h3>
            <p className="text-sm text-[var(--color-text-muted)] mb-4">
              Dashboard API 端点配置
            </p>

            <div className="space-y-4">
              <div>
                <label className="block text-sm text-[var(--color-text-secondary)] mb-1">
                  API Base URL
                </label>
                <input
                  type="text"
                  value={import.meta.env.VITE_API_URL || '/v1'}
                  disabled
                  className="w-full px-3 py-2 bg-[var(--color-background)] border border-[var(--color-border)] rounded-lg text-sm text-[var(--color-text-muted)] cursor-not-allowed"
                />
                <p className="text-xs text-[var(--color-text-muted)] mt-1">
                  通过环境变量 VITE_API_URL 配置
                </p>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
