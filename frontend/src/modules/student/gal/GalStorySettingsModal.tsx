import { Form, Modal, Radio, Space, Typography } from 'antd'

import { useI18n } from '@/core/i18n/useI18n'
import type { GalGameSettings } from '@/modules/student/gal/galTypes'

export function GalStorySettingsModal({
  open,
  settings,
  onCancel,
  onConfirm,
}: {
  open: boolean
  settings: GalGameSettings
  onCancel: () => void
  onConfirm: (next: GalGameSettings) => void
}) {
  const { t } = useI18n()
  const [form] = Form.useForm<GalGameSettings>()

  return (
    <Modal
      title={t('student.gal.settingsTitle')}
      open={open}
      okText={t('student.gal.startGenerate')}
      cancelText={t('student.gal.cancel')}
      onCancel={onCancel}
      onOk={() => {
        form.validateFields().then((values) => onConfirm(values))
      }}
      destroyOnClose
    >
      <Typography.Paragraph type="secondary">{t('student.gal.settingsHint')}</Typography.Paragraph>
      <Form form={form} layout="vertical" initialValues={settings}>
        <Form.Item name="detailLevel" label={t('student.gal.detailLevel')}>
          <Radio.Group>
            <Space direction="vertical">
              <Radio value="brief">{t('student.gal.detailBrief')}</Radio>
              <Radio value="detailed">{t('student.gal.detailDetailed')}</Radio>
              <Radio value="academic">{t('student.gal.detailAcademic')}</Radio>
            </Space>
          </Radio.Group>
        </Form.Item>
        <Form.Item name="personality" label={t('student.gal.personality')}>
          <Radio.Group>
            <Space direction="vertical">
              <Radio value="gentle">{t('student.gal.personalityGentle')}</Radio>
              <Radio value="tsundere">{t('student.gal.personalityTsundere')}</Radio>
              <Radio value="strict">{t('student.gal.personalityStrict')}</Radio>
            </Space>
          </Radio.Group>
        </Form.Item>
      </Form>
    </Modal>
  )
}
