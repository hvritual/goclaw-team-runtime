# Space

素材（Space）是素材对象及其存储生命周期的上下文。素材始终按工作区隔离，但它被用作附件、项目资源或知识证据的业务含义由消费上下文拥有。

## Language

**Asset**:
具有稳定身份、媒体类型、大小、校验信息和访问策略的工作区素材对象。
_Avoid_: Attachment、Project Resource、Knowledge Entry

**Asset Version**:
Asset 内容的一次不可变修订。
_Avoid_: Skill Version、Document Revision

**Asset Reference**:
其他上下文保存的稳定素材 ID；引用本身不转移业务所有权。
_Avoid_: Copied Asset Metadata、Shared Database Row

**Storage Object**:
承载素材内容的底层对象，不具备项目、issue 或知识语义。
_Avoid_: Asset、Attachment
